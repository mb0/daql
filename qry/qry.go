package qry

import (
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

type Backend interface {
	Eval(*exp.Ctx, exp.Env, *Doc) (lit.Lit, error)
}

func (p *Doc) Find(name string) *Task {
	for _, t := range p.Root {
		if t.Name == name {
			return t
		}
	}
	return nil
}

func (p *Doc) Resolve(c *exp.Ctx, env exp.Env, x *exp.Call, hint typ.Type) (exp.El, error) {
	return x, nil
}
func (p *Doc) Execute(c *exp.Ctx, env exp.Env, x *exp.Call, hint typ.Type) (exp.El, error) {
	qenv := FindEnv(env)
	if qenv == nil && qenv.Backend == nil {
		return nil, cor.Errorf("no qry backend configured for query %s", x)
	}
	var arg lit.Lit = lit.Nil
	if a, ok := x.Arg(0).(*exp.Atom); ok {
		arg = a.Lit
	}
	res, err := qenv.Backend.Eval(c, &exp.ParamEnv{env, arg}, p)
	if err != nil {
		return nil, err
	}
	return &exp.Atom{Lit: res}, nil
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
