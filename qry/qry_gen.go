// generated code

package qry

import (
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/typ"
)

// Type is the task's result type or void if not yet resolved.
type Task struct {
	Name   string   `json:"name"`
	Expr   exp.El   `json:"expr,omitempty"`
	Query  *Query   `json:"query,omitempty"`
	Parent *Task    `json:"parent,omitempty"`
	Type   typ.Type `json:"type,omitempty"`
}

type Ord struct {
	Key  string `json:"key"`
	Desc bool   `json:"desc,omitempty"`
}

type Query struct {
	Ref  string   `json:"ref"`
	Type typ.Type `json:"type"`
	Whr  *exp.Dyn `json:"whr,omitempty"`
	Ord  []Ord    `json:"ord,omitempty"`
	Off  int64    `json:"off,omitempty"`
	Lim  int64    `json:"lim,omitempty"`
	Sel  []*Task  `json:"sel,omitempty"`
	Sca  bool     `json:"sca,omitempty"`
}

// Doc represents a whole query document, consisting of one or more tasks.
type Doc struct {
	Root []*Task  `json:"root"`
	Type typ.Type `json:"type,omitempty"`
}
