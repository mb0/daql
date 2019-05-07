// generated code

package evt

import (
	"time"

	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
)

type Detail struct {
	Created time.Time `json:"created,omitempty"`
	Arrived time.Time `json:"arrived,omitempty"`
	Acct    [16]byte  `json:"acct,omitempty"`
	Extra   *lit.Dict `json:"extra,omitempty"`
}

type Audit struct {
	Rev time.Time `json:"rev"`
	Detail
}

type Sig struct {
	Top string `json:"top"`
	Key string `json:"key"`
}

type Action struct {
	Sig
	Cmd string    `json:"cmd"`
	Arg *lit.Dict `json:"arg,omitempty"`
}

type Event struct {
	ID  int64     `json:"id"`
	Rev time.Time `json:"rev"`
	Action
}

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

type Result struct {
	Rev time.Time `json:"rev"`
	Val lit.Lit   `json:"val"`
}

type QryReq struct {
	Arg exp.Dyn `json:"arg"`
}

type QryRes struct {
	Res *Result `json:"res,omitempty"`
	Err string  `json:"err,omitempty"`
}

type MetaReq struct {
	Rev time.Time `json:"rev"`
}

type MetaRes struct {
	Res *Audit `json:"res,omitempty"`
	Err string `json:"err,omitempty"`
}

type HistReq struct {
	Sig
}

type HistRes struct {
	Res *Update `json:"res,omitempty"`
	Err string  `json:"err,omitempty"`
}

type PubReq struct {
	Trans
}

type PubRes struct {
	Res *Update `json:"res,omitempty"`
	Err string  `json:"err,omitempty"`
}

type SubReq struct {
	List []Watch `json:"list"`
}

type SubRes struct {
	Res *Update `json:"res,omitempty"`
	Err string  `json:"err,omitempty"`
}

type UnsubReq struct {
	List []Watch `json:"list"`
}

type UnsubRes struct {
	Res bool   `json:"res,omitempty"`
	Err string `json:"err,omitempty"`
}
