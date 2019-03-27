// generated code

package evt

import (
	"github.com/mb0/xelf/lit"
	"time"
)

type Audit struct {
	Rev     time.Time `json:"rev"`
	Created time.Time `json:"created"`
	Arrived time.Time `json:"arrived"`
	Acct    [16]byte  `json:"acct,omitempty"`
	Extra   *lit.Dict `json:"extra,omitempty"`
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

type Update struct {
	Rev time.Time `json:"rev"`
	Evs []*Event  `json:"evs"`
}

type Pub struct {
	Base    time.Time `json:"base"`
	Actions []Action  `json:"actions"`
	Created time.Time `json:"created,omitempty"`
	Extra   *lit.Dict `json:"extra,omitempty"`
}

type Trans struct {
	Pub
	Acct    [16]byte  `json:"acct,omitempty"`
	Arrived time.Time `json:"arrived"`
}
