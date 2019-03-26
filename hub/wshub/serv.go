package wshub

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mb0/daql/hub"
)

func Serve(h *hub.Hub) http.HandlerFunc {
	upgr := &websocket.Upgrader{}
	return func(w http.ResponseWriter, r *http.Request) {
		wc, err := upgr.Upgrade(w, r, nil)
		if err != nil {
			// TODO introduce an abstract logger interface to daql
			log.Printf("hub ws upgrade failed, err: %v", err)
			return
		}
		c := &conn{id: hub.NextID(), wc: wc, route: h.Chan(), send: make(chan *hub.Msg, 32)}
		t := time.NewTicker(60 * time.Second)
		defer t.Stop()
		hub.Signon(h, c)
		go write(c, t)
		err = c.read()
		hub.Signoff(h, c)
		if err != nil {
			// TODO introduce an abstract logger interface to daql
			log.Printf("hub ws read failed, err: %v", err)
		}
	}
}

func write(c *conn, t *time.Ticker) {
	defer c.wc.Close()
Outer:
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				break Outer
			}
			err := c.writeMsg(msg)
			if err != nil {
				// TODO log marshal errors but ignore network errors.
				return
			}
		case <-t.C:
			c.wc.SetWriteDeadline(time.Now().Add(writeTimeout))
			err := c.wc.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				return // ignore error
			}
		}
	}
	c.wc.SetWriteDeadline(time.Now().Add(writeTimeout))
	c.wc.WriteMessage(websocket.CloseMessage, []byte{})
}
