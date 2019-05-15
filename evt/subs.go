package evt

import (
	"time"

	"github.com/mb0/daql/hub"
)

type Subscriber struct {
	hub.Conn
	Rev   time.Time
	Watch map[string]*Watch
	Bufr  []*Event
}

func (s *Subscriber) Accept(ev *Event) bool {
	w := s.Watch[ev.Top]
	if w == nil {
		return false
	}
	if !ev.Rev.After(w.Rev) {
		return false
	}
	if len(w.IDs) > 0 {
		for _, id := range w.IDs {
			if ev.Key == id {
				return true
			}
		}
		return false
	}
	return true
}

func (s *Subscriber) Update(from hub.Conn, rev time.Time) {
	s.Rev = rev
	res := Update{Rev: rev}
	if len(s.Bufr) > 0 {
		res.Evs = make([]*Event, len(s.Bufr))
		copy(res.Evs, s.Bufr)
		s.Bufr = s.Bufr[:0]
	}
	s.Chan() <- &hub.Msg{From: from, Subj: "update", Data: res}
}

type Subscribers struct {
	smap  map[int64]*Subscriber
	tmap  map[string][]*Subscriber
	btrig *time.Timer
	bcast time.Time
}

func NewSubscribers() *Subscribers {
	return &Subscribers{
		smap: make(map[int64]*Subscriber),
		tmap: make(map[string][]*Subscriber),
	}
}

func (subs *Subscribers) Show(c hub.Conn, evs []*Event) (sender *Subscriber) {
	id := c.ID()
	for _, ev := range evs {
		for _, s := range subs.tmap[ev.Top] {
			if s.ID() == id {
				sender = s
			} else if s.Accept(ev) {
				s.Bufr = append(s.Bufr, ev)
			}
		}
	}
	return sender
}

func (subs *Subscribers) Sub(c hub.Conn, ws []Watch) *Subscriber {
	if len(ws) == 0 {
		return nil
	}
	id := c.ID()
	s := subs.smap[id]
	if s == nil {
		s = &Subscriber{Conn: c, Watch: make(map[string]*Watch)}
		subs.smap[id] = s
	}
	for _, w := range ws {
		o := s.Watch[w.Top]
		if o != nil {
			// TODO merge watch
		} else {
			subs.tmap[w.Top] = append(subs.tmap[w.Top], s)
		}
		s.Watch[w.Top] = &w

	}
	s.Bufr = filter(s.Bufr, ws)
	return s
}

func (subs *Subscribers) Unsub(c hub.Conn, ws []Watch) {
	id := c.ID()
	if len(ws) == 0 {
		delete(subs.smap, id)
		return
	}
	s := subs.smap[id]
	if s == nil {
		return
	}
	for _, w := range ws {
		// TODO detangle watch
		delete(s.Watch, w.Top)
		list := subs.tmap[w.Top]
		for i, el := range list {
			if s == el {
				subs.tmap[w.Top] = append(list[:i], list[i+1:]...)
				break
			}
		}
	}
	if len(s.Watch) == 0 {
		delete(subs.smap, id)
	}
	s.Bufr = filter(s.Bufr, ws)
}

// Btrig trigger fires a delayed, de-duped broadcast request with header _bcast.
func (subs *Subscribers) Btrig(from hub.Conn) {
	if subs.btrig != nil && time.Now().Sub(subs.bcast) < 2*time.Second {
		subs.btrig.Stop()
	}
	subs.btrig = time.AfterFunc(200*time.Millisecond, func() {
		from.Chan() <- &hub.Msg{From: from, Subj: "_bcast"}
	})
}

// Bcast sends all buffered events up to revision rev out to subscribers.
func (subs *Subscribers) Bcast(from hub.Conn, rev time.Time) {
	if !rev.After(subs.bcast) {
		return
	}
	subs.bcast = rev
	for _, s := range subs.smap {
		s.Update(from, rev)
	}
}

func (subs *Subscribers) Stop() {
	if subs.btrig != nil {
		subs.btrig.Stop()
	}
}

func filter(evs []*Event, subs []Watch) []*Event {
	out := evs[:0] // reuse
Outer:
	for _, ev := range evs {
		for _, sub := range subs {
			if sub.Top == ev.Top {
				continue Outer
			}
		}
		out = append(out, ev)
	}
	return out
}
