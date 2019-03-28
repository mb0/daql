package wshub

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/mb0/daql/hub"
	"github.com/mb0/daql/log"
)

type TokenProvider interface {
	Token(url string) (http.Header, error)
	ClearToken(url string) error
}

type Client struct {
	url  string
	id   int64
	send chan *hub.Msg
	*websocket.Dialer
	TokenProvider
	Log log.Logger
}

func NewClient(url string) *Client {
	return &Client{url: url, id: hub.NextID(), send: make(chan *hub.Msg, 32)}
}

func (c *Client) ID() int64             { return c.id }
func (c *Client) Chan() chan<- *hub.Msg { return c.send }

func (c *Client) Connect(r chan<- *hub.Msg) error {
	c.init()
	hdr, err := c.Token(c.url)
	if err != nil {
		return err
	}
	wc, _, err := c.Dial(c.url, hdr)
	if err != nil {
		c.ClearToken(c.url)
		return err
	}
	cc := newConn(c.id, wc, c.send)
	r <- &hub.Msg{From: c, Subj: hub.SubjSignon}
	go cc.writeAll(c.id, c.Log)
	err = cc.readAll(r)
	r <- &hub.Msg{From: c, Subj: hub.SubjSignoff}
	return err
}

func (c *Client) init() {
	if c.Dialer == nil {
		c.Dialer = websocket.DefaultDialer
	}
	if c.Log == nil {
		c.Log = log.Root
	}
	if c.TokenProvider == nil {
		c.TokenProvider = (*nilProvider)(nil)
	}
}

type nilProvider struct{}

func (*nilProvider) Token(string) (http.Header, error) { return nil, nil }
func (*nilProvider) ClearToken(string) error           { return nil }
