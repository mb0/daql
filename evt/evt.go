// generated code

package evt

import (
	"time"
)

type Audit struct {
	Rev     time.Time              `json:"rev"`
	Created time.Time              `json:"created"`
	Arrived time.Time              `json:"arrived"`
	Acct    [16]byte               `json:"acct,omitempty"`
	Extra   map[string]interface{} `json:"extra"`
}

type Action struct {
	Top string                 `json:"top"`
	Key string                 `json:"key"`
	Cmd string                 `json:"cmd"`
	Arg map[string]interface{} `json:"arg"`
}

type Event struct {
	ID  int64     `json:"id"`
	Rev time.Time `json:"rev"`
	Action
}

type Pub struct {
	Base    time.Time              `json:"base"`
	Actions []Action               `json:"actions"`
	Created *time.Time             `json:"created"`
	Extra   map[string]interface{} `json:"extra"`
}

type Trans struct {
	Pub
	Acct    *[16]byte `json:"acct,omitempty"`
	Arrived time.Time `json:"arrived"`
}
