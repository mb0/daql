package wshub

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mb0/daql/hub"
	"github.com/mb0/daql/log"
	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/cor"
)

type conn struct {
	id   int64
	wc   *websocket.Conn
	send chan *hub.Msg
	tick <-chan time.Time
}

func newConn(id int64, wc *websocket.Conn, send chan *hub.Msg) *conn {
	if send == nil {
		send = make(chan *hub.Msg, 32)
	}
	return &conn{id: id, wc: wc, send: send}
}

func (c *conn) ID() int64             { return c.id }
func (c *conn) Chan() chan<- *hub.Msg { return c.send }

func (c *conn) readAll(route chan<- *hub.Msg) error {
	for {
		op, r, err := c.wc.NextReader()
		if err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				return nil // ignore error client disconnected
			}
			if cerr, ok := err.(*websocket.CloseError); ok && cerr.Code == 1001 {
				return nil // ignore error client disconnected
			}
			return cor.Errorf("wshub client next reader: %w", err)
		}
		if op == websocket.BinaryMessage {
			return cor.Errorf("wshub client unexpected binary message: %w", err)
		}
		if op != websocket.TextMessage {
			continue
		}
		m, err := readMsg(r)
		if err != nil {
			return cor.Errorf("wshub msg read failed: %w", err)
		}
		m.From = c
		route <- m
	}
}

func (c *conn) writeAll(id int64, log log.Logger) {
	defer c.wc.Close()
	for {
		select {
		case m := <-c.send:
			if m == nil {
				c.write(websocket.CloseMessage, []byte{}, time.Second)
				return
			}
			err := c.writeMsg(m, 20*time.Second, log)
			if err != nil {
				return
			}
		case <-c.tick:
			err := c.write(websocket.PingMessage, []byte{}, 5*time.Second)
			if err != nil {
				return
			}
		}
	}
}

func readMsg(r io.Reader) (*hub.Msg, error) {
	b := bfr.Get()
	defer bfr.Put(b)

	_, err := b.ReadFrom(r)
	if err != nil {
		return nil, err
	}
	var tok, body []byte
	head := b.Bytes()
	idx := bytes.IndexByte(head, '\n')
	if idx >= 0 {
		head, body = head[:idx], head[idx+1:]
	}
	idx = bytes.IndexByte(head, '#')
	if idx >= 0 {
		head, tok = head[:idx], head[idx+1:]
	}
	if len(head) == 0 {
		return nil, cor.Error("message without subject")
	}
	return &hub.Msg{
		Subj: string(head),
		Tok:  copyBytes(tok),
		Raw:  copyBytes(body),
	}, nil
}

func (c *conn) write(kind int, data []byte, timeout time.Duration) error {
	c.wc.SetWriteDeadline(time.Now().Add(timeout))
	return c.wc.WriteMessage(kind, data)
}

func (c *conn) writeMsg(msg *hub.Msg, timeout time.Duration, log log.Logger) error {
	b := bfr.Get()
	defer bfr.Put(b)
	err := writeMsgTo(b, msg)
	if err != nil {
		log.Error("write msg", "err", err)
		return err
	}
	return c.write(websocket.TextMessage, b.Bytes(), timeout)
}

func writeMsgTo(b bfr.B, m *hub.Msg) error {
	_, err := b.WriteString(m.Subj)
	if err != nil {
		return err
	}
	if len(m.Tok) != 0 {
		b.WriteByte('#')
		_, err = b.Write(m.Tok)
		if err != nil {
			return err
		}
	}
	if len(m.Raw) != 0 {
		b.WriteByte('\n')
		_, err = b.Write(m.Raw)
		return err
	}
	if m.Data != nil {
		b.WriteByte('\n')
		if w, ok := m.Data.(bfr.Writer); ok {
			return w.WriteBfr(&bfr.Ctx{B: b, JSON: true})
		} else {
			return json.NewEncoder(b).Encode(m.Data)
		}
	}
	return nil
}

func copyBytes(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	res := make([]byte, len(b))
	copy(res, b)
	return res
}
