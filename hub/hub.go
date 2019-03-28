// Package hub provides a transport agnostic connection hub.
package hub

import "sync"

const (
	SubjSignon  = "+"
	SubjSignoff = "-"
)

// Msg is the central data structure passed between connections.
//
// The from and subj fields must be populated. Tok can be used by the origin connection to match
// replies to requests, is otherwise unprocessed and completely optional. Message body can be
// optional, depending on the message subject, and is either represented by raw bytes or a typed
// data, or both. The body type should not vary for the same subject, other than between request
// and replies. If raw is not populated and data is, a transport may choose a serialization format,
// usually JSON. If raw is populated and data isn't, the connection that ultimately knows how to
// handle messages of that subject parses the data. The data field can effectively used to avoid
// message body serialization to and from bytes for in-process messages.
type Msg struct {
	// From is the connection this message originates from.
	From Conn
	// Subj is the message header used for routing and determining the data type.
	Subj string
	Tok  []byte
	Raw  []byte
	Data interface{}
}

// Router routes a received message to connection.
type Router interface{ Route(*Msg) }

// Conn is the common interface providing an ID and channel for participants connected to a hub.
//
// Connections can represent one-off calls, connected clients of any kind, the hub itself or
// services like chat rooms. Connections can hold on to received message sender connections.
type Conn interface {
	// ID is an internal connection identifier, the hub has id 0, transient connections have a
	// negative and normal connections positive ids.
	ID() int64
	// Chan returns an unchanging receiver channel. The hub send a nil message to this
	// channel after a sign-off message from this conn was routed.
	Chan() chan<- *Msg
}

// Hub is the central server participant that manages connection sign-on and sign-offs and keeps a
// list of all signed on participant. Hub itself implements a Conn with ID 0.
//
// One-off connections used for a simple request-response round trips can be used without sign-on
// and must use the special ID -1. These connections can only responded to directly and must not be
// hold on to. The acceptors that send messages to hub for routing are also responsible for sender
// sign-on and validation.
type Hub struct {
	sync.Mutex
	cmap map[int64]Conn
	mque chan *Msg
}

// NewHub creates and returns a new hub.
func NewHub() *Hub {
	return &Hub{
		cmap: make(map[int64]Conn, 64),
		mque: make(chan *Msg, 128),
	}
}

func (h *Hub) ID() int64         { return 0 }
func (h *Hub) Chan() chan<- *Msg { return h.mque }

// Run starts routing received messages with the given router. It is usually run in a go routine.
func (h *Hub) Run(r Router) {
	for m := range h.mque {
		if m == nil {
			break
		}
		if m.Subj == SubjSignon {
			h.Lock()
			h.cmap[m.From.ID()] = m.From
			h.Unlock()
		}
		r.Route(m)
		if m.Subj == SubjSignoff {
			h.Lock()
			delete(h.cmap, m.From.ID())
			m.From.Chan() <- nil
			h.Unlock()
		}
	}
}
