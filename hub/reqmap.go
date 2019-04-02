package hub

import (
	"strconv"

	"github.com/mb0/xelf/cor"
)

type RequestMap struct {
	last int64
	m    map[int64]req
}

func (r *RequestMap) Note(m *Msg) []byte {
	r.last++
	if r.m == nil {
		r.m = make(map[int64]req)
	}
	r.m[r.last] = req{m.From, m.Tok}
	return strconv.AppendInt(nil, r.last, 16)
}

func (r *RequestMap) Response(m *Msg) error {
	if len(m.Tok) == 0 {
		return cor.Errorf("empty token for response %s", m.Subj)
	}
	tok := string(m.Tok)
	id, err := strconv.ParseInt(tok, 16, 64)
	if err != nil {
		return cor.Errorf("invalid token %s: %w", tok, err)
	}
	req, ok := r.m[id]
	if !ok {
		return cor.Errorf("no request with token %s", tok)
	}
	n := *m
	n.Tok = req.tok
	req.Chan() <- &n
	delete(r.m, id)
	return nil
}

type req struct {
	Conn
	tok []byte
}
