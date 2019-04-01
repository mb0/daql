// generated code

package evt

import (
	"github.com/mb0/xelf/lit"
	"time"
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
