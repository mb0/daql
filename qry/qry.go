package qry

import (
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
)

// Collect returns a slice of all query tasks and all root task that may not be queries.
func Collect(pl *Plan) []*Task {
	res := make([]*Task, 0, len(pl.Root)*2)
	return collectTasks(res, pl.Root, true)
}

func collectTasks(dst, src []*Task, root bool) []*Task {
	for _, t := range src {
		if t.Query != nil {
			dst = append(dst, t)
			dst = collectTasks(dst, t.Query.Sel, false)
		} else if root {
			dst = append(dst, t)
		}
	}
	return dst
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
