package qry

import (
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

type Backend interface {
	Exec(*exp.Ctx, exp.Env, *Doc) (*Result, error)
}

// TaskInfo holds task details during query execution.
// Done indicates whether the task and all its sub task are represented by data.
type TaskInfo struct {
	Data lit.Proxy
	Done bool
}

type Result struct {
	Data lit.Proxy
	Info map[*Task]TaskInfo
}

func NewResult(d *Doc) *Result {
	t, opt := d.Type.Deopt()
	data := lit.ZeroProxy(t)
	if opt {
		data = lit.SomeProxy{data}
	}
	return &Result{data, make(map[*Task]TaskInfo, len(d.Root)*3)}
}

func (r *Result) Prep(parent lit.Proxy, t *Task) (lit.Proxy, error) {
	if t.Name == "" {
		return parent, nil
	}
	k := lit.Deopt(parent).(lit.Keyer)
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

func (r *Result) SetDone(t *Task, val lit.Proxy) {
	n := r.Info[t]
	n.Data = val
	n.Done = true
	r.Info[t] = n
}
func (r *Result) IsDone(t *Task) bool {
	return r.Info[t].Done
}

func (p *Doc) Find(name string) *Task {
	for _, t := range p.Root {
		if t.Name == name {
			return t
		}
	}
	return nil
}

func RootTask(p *Doc, path string) (*Task, lit.Path, error) {
	if path == "" || path == "/" {
		return nil, nil, cor.Errorf("task not found %s", path)
	}
	if path[0] == '/' {
		path = path[1:]
	}
	lp, err := lit.ReadPath(path)
	if err != nil {
		return nil, nil, err
	}
	t := p.Find(lp[0].Key)
	if t == nil {
		return nil, lp, cor.Errorf("task not found %s", path)
	}
	return t, lp[1:], nil
}

func (p *Doc) Resolve(c *exp.Ctx, env exp.Env, x *exp.Call, hint typ.Type) (exp.El, error) {
	if !c.Exec {
		return x, exp.ErrExec
	}
	qenv := FindEnv(env)
	if qenv == nil && qenv.Backend == nil {
		return nil, cor.Errorf("no qry backend configured for query %s", x)
	}
	res, err := qenv.Backend.Exec(c, env, p)
	if err != nil {
		return nil, err
	}
	return &exp.Atom{Lit: res.Data}, nil
}
