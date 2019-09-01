package qry

import (
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

type Backend interface {
	Exec(*exp.Prog, exp.Env, *Doc) (lit.Lit, error)
}

func (d *Doc) Find(name string) *Task {
	for _, t := range d.Root {
		if t.Name == name {
			return t
		}
	}
	return nil
}

func (d *Doc) Resl(p *exp.Prog, env exp.Env, c *exp.Call, h typ.Type) (exp.El, error) {
	return c, nil
}
func (d *Doc) Eval(p *exp.Prog, env exp.Env, c *exp.Call, h typ.Type) (exp.El, error) {
	qenv := FindEnv(env)
	if qenv == nil && qenv.Backend == nil {
		return nil, cor.Errorf("no qry backend configured for query %s", c)
	}
	var arg lit.Lit = lit.Nil
	if a, ok := c.Arg(0).(*exp.Atom); ok {
		arg = a.Lit
	}
	res, err := qenv.Backend.Exec(p, &exp.ParamEnv{env, arg}, d)
	if err != nil {
		return nil, err
	}
	return &exp.Atom{Lit: res}, nil
}

func RootTask(d *Doc, path string) (*Task, lit.Path, error) {
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
	t := d.Find(lp[0].Key)
	if t == nil {
		return nil, lp, cor.Errorf("task not found %s", path)
	}
	return t, lp[1:], nil
}
