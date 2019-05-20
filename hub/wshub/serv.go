// Package wshub provides a websocket server and client using gorilla/websocket for package hub.
package wshub

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mb0/daql/hub"
	"github.com/mb0/daql/log"
)

type Server struct {
	*hub.Hub
	*websocket.Upgrader
	Log log.Logger
}

func (s *Server) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	s.init()
	wc, err := s.Upgrade(w, r, nil)
	if err != nil {
		s.Log.Error("wshub upgrade failed", "err", err)
		return
	}
	route := s.Chan()
	t := time.NewTicker(60 * time.Second)
	defer t.Stop()
	c := newConn(hub.NextID(), wc, nil)
	c.tick = t.C
	route <- &hub.Msg{From: s.Hub, Subj: hub.SubjSignon}
	go c.writeAll(0, s.Log)
	err = c.readAll(route)
	route <- &hub.Msg{From: s.Hub, Subj: hub.SubjSignoff}
	if err != nil {
		s.Log.Error("wshub read failed", "err", err)
	}
}

func (s *Server) init() {
	if s.Upgrader == nil {
		s.Upgrader = &websocket.Upgrader{}
	}
	if s.Log == nil {
		s.Log = log.Root
	}
}
