package qry

import (
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

// Plan represents a whole query request, consisting of one or more tasks.
type Plan struct {
	Root []*Task
	Type typ.Type
}

// Task is a unit of work as a part of a greater query plan. Root tasks are either expression or
// query tasks. Expression tasks consist of a xelf expressions, that cannot query the data source,
// but can reference results of previous tasks. Query tasks do access the data source, and may have
// a list of explicit selection tasks. The selection tasks can have only simple field names or an
// expression or sub query. In effect building a tree of queries.
type Task struct {
	Name  string
	Expr  exp.El
	Query *Query

	// Type is the task's result type or void if not yet resolved.
	Type typ.Type
}

type Ord struct {
	Key  string
	Desc bool
}

type Query struct {
	Ref string
	// Type represents the query subject type.
	Type typ.Type
	// Whr is a list of expression elements treated as 'and' arguments.
	// The whr clause can only refer to full subject but none of the extra selections.
	Whr exp.Dyn
	// Ord is a list of result symbols used for ordering. We may at some point allow references
	// to the subject and not only the selection, like sql does in the ordering clause.
	Ord []Ord
	Off int
	Lim int
	Sel []*Task
}

func Prep(pa lit.Proxy, t *Task) (lit.Proxy, error) {
	if t.Name == "" {
		return pa, nil
	}
	k := lit.Deopt(pa).(lit.Keyer)
	l, err := k.Key(cor.Keyed(t.Name))
	if err != nil {
		return nil, err
	}
	p, ok := l.(lit.Proxy)
	if !ok {
		return nil, cor.Errorf("prep task result for %s want proxy got %T", t.Name, l)
	}
	return p, nil
}

// TaskInfo holds task details during query execution.
// Done indicatates whether the task and all its sub task are represented by data.
type TaskInfo struct {
	Parent *Task
	Data   lit.Proxy
	Path   lit.Path
	Done   bool
}

type Result struct {
	Data lit.Proxy
	Done bool
	Info map[*Task]TaskInfo
}

type Ctx struct {
	*exp.Ctx
	*Plan
	Result
}

func NewCtx(c *exp.Ctx, p *Plan) *Ctx {
	t, opt := p.Type.Deopt()
	data := lit.ZeroProxy(t)
	if opt {
		data = lit.SomeProxy{data}
	}
	return &Ctx{c, p, Result{data, false, make(map[*Task]TaskInfo, len(p.Root)*3)}}
}
func (c *Ctx) SetDone(t *Task, val lit.Proxy) {
	n := c.Info[t]
	n.Data = val
	n.Done = true
	c.Info[t] = n
}

type Backend interface {
	ExecPlan(*Ctx, exp.Env) error
}
