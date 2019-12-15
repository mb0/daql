// generated code

package evt

import (
	"encoding/json"
	"time"

	"github.com/mb0/daql/hub"
	"github.com/mb0/xelf/lit"
)

// Detail holds extra information for audits and translations.
type Detail struct {
	Created time.Time `json:"created,omitempty"`
	Arrived time.Time `json:"arrived,omitempty"`
	Acct    [16]byte  `json:"acct,omitempty"`
	Extra   *lit.Dict `json:"extra,omitempty"`
}

// Audit holds detailed information for a published revision.
type Audit struct {
	Rev time.Time `json:"rev"`
	Detail
}

// Sig is the event signature.
type Sig struct {
	Top string `json:"top"`
	Key string `json:"key"`
}

// Action is an unpublished event represented by a command string and argument map.
// It usually is a data operation on a record identified by a topic and primary key.
type Action struct {
	Sig
	Cmd string    `json:"cmd"`
	Arg *lit.Dict `json:"arg,omitempty"`
}

// Event is an action published to a ledger with revision and unique id.
type Event struct {
	ID  int64     `json:"id"`
	Rev time.Time `json:"rev"`
	Action
}

// Trans is an request to publish a list of actions for a base revision.
type Trans struct {
	Base time.Time `json:"base"`
	Acts []Action  `json:"acts"`
	Detail
}

type Watch struct {
	Top string    `json:"top"`
	Rev time.Time `json:"rev,omitempty"`
	IDs []string  `json:"ids,omitempty"`
}

type Update struct {
	Rev time.Time `json:"rev"`
	Evs []*Event  `json:"evs"`
}

type MetaReq struct {
	Rev time.Time `json:"rev"`
}

type MetaRes struct {
	Res *Audit `json:"res,omitempty"`
	Err string `json:"err,omitempty"`
}

type MetaFunc func(*hub.Msg, MetaReq) (*Audit, error)

func (f MetaFunc) Serve(m *hub.Msg) interface{} {
	var req MetaReq
	err := json.Unmarshal(m.Raw, &req)
	if err != nil {
		return MetaRes{Err: err.Error()}
	}
	res, err := f(m, req)
	if err != nil {
		return MetaRes{Err: err.Error()}
	}
	return MetaRes{Res: res}
}

type HistReq struct {
	Sig
}

type HistRes struct {
	Res *Update `json:"res,omitempty"`
	Err string  `json:"err,omitempty"`
}

type HistFunc func(*hub.Msg, HistReq) (*Update, error)

func (f HistFunc) Serve(m *hub.Msg) interface{} {
	var req HistReq
	err := json.Unmarshal(m.Raw, &req)
	if err != nil {
		return HistRes{Err: err.Error()}
	}
	res, err := f(m, req)
	if err != nil {
		return HistRes{Err: err.Error()}
	}
	return HistRes{Res: res}
}

type PubReq struct {
	Trans
}

type PubRes struct {
	Res *Update `json:"res,omitempty"`
	Err string  `json:"err,omitempty"`
}

type PubFunc func(*hub.Msg, PubReq) (*Update, error)

func (f PubFunc) Serve(m *hub.Msg) interface{} {
	var req PubReq
	err := json.Unmarshal(m.Raw, &req)
	if err != nil {
		return PubRes{Err: err.Error()}
	}
	res, err := f(m, req)
	if err != nil {
		return PubRes{Err: err.Error()}
	}
	return PubRes{Res: res}
}

type SubReq struct {
	List []Watch `json:"list"`
}

type SubRes struct {
	Res *Update `json:"res,omitempty"`
	Err string  `json:"err,omitempty"`
}

type SubFunc func(*hub.Msg, SubReq) (*Update, error)

func (f SubFunc) Serve(m *hub.Msg) interface{} {
	var req SubReq
	err := json.Unmarshal(m.Raw, &req)
	if err != nil {
		return SubRes{Err: err.Error()}
	}
	res, err := f(m, req)
	if err != nil {
		return SubRes{Err: err.Error()}
	}
	return SubRes{Res: res}
}

type UnsReq struct {
	List []Watch `json:"list"`
}

type UnsRes struct {
	Res bool   `json:"res,omitempty"`
	Err string `json:"err,omitempty"`
}

type UnsFunc func(*hub.Msg, UnsReq) (bool, error)

func (f UnsFunc) Serve(m *hub.Msg) interface{} {
	var req UnsReq
	err := json.Unmarshal(m.Raw, &req)
	if err != nil {
		return UnsRes{Err: err.Error()}
	}
	res, err := f(m, req)
	if err != nil {
		return UnsRes{Err: err.Error()}
	}
	return UnsRes{Res: res}
}
