// generated code

package qry

import (
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/typ"
)

// Task is a unit of work as a part of a greater query plan. Root tasks are either expression or
// query tasks. Expression tasks consist of a xelf expressions, that cannot query the data source,
// but can reference results of previous tasks. Query tasks do access the data source, and may have
// a list of explicit selection tasks. The selection tasks can have only simple field names or an
// expression or sub query. In effect building a tree of queries.
type Task struct {
	Name  string   `json:"name"`
	Expr  exp.El   `json:"expr,omitempty"`
	Query *Query   `json:"query,omitempty"`
	Type  typ.Type `json:"type,omitempty"`
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
}

// Plan represents a whole query request, consisting of one or more tasks.
type Plan struct {
	Root []*Task  `json:"root"`
	Type typ.Type `json:"type,omitempty"`
}
