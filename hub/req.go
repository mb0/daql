package hub

import (
	"sync/atomic"
	"time"

	"github.com/mb0/xelf/cor"
)

// lastID holds the last id returned from next id. It must only be accessed as atomic primitives.
var lastID = new(int64)

// NextID returns a new unused normal connection id.
func NextID() int64 { return int64(atomic.AddInt64(lastID, 1)) }

// ChanConn is a channel based connection used for simple in-process hub participants.
type ChanConn struct {
	id int64
	ch chan *Msg
}

// NewChanConn returns a new channel connection with the given id and channel.
func NewChanConn(id int64, c chan *Msg) *ChanConn { return &ChanConn{id, c} }

func (c *ChanConn) ID() int64         { return c.id }
func (c *ChanConn) Chan() chan<- *Msg { return c.ch }

// Req sends req to the hub from a newly created transient connection and returns the first response
// or an error if the timeout was reached.
func Req(h Hub, req *Msg, timeout time.Duration) (*Msg, error) {
	ch := make(chan *Msg, 1)
	c := NewChanConn(-1, ch)
	req.From = c
	h.Chan() <- req
	select {
	case res := <-ch:
		if res == nil {
			return nil, cor.Error("conn closed")
		}
		return res, nil
	case <-time.After(timeout):
	}
	return nil, cor.Error("timeout")
}
