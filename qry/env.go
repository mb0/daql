package qry

import (
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
	"github.com/mb0/xelf/utl"
)

var Builtin = exp.Builtin{
	QryLookup,
	utl.StrLib.Lookup(),
	utl.TimeLib.Lookup(),
	exp.Std, exp.Core,
}

func NewEnv(env exp.Env, bend Backend) *PlanEnv {
	if env == nil {
		env = Builtin
	}
	s := &exp.ParamScope{exp.NewScope(env), nil}
	return &PlanEnv{s, &Plan{}, bend}
}

func QryLookup(sym string) exp.Resolver {
	if sym == "qry" {
		return qryForm
	}
	return nil
}

type PlanEnv struct {
	Par exp.Env
	*Plan
	Backend
}

func (p *PlanEnv) Parent() exp.Env              { return p.Par }
func (*PlanEnv) Supports(x byte) bool           { return x == '/' }
func (*PlanEnv) Def(string, exp.Resolver) error { return exp.ErrNoDefEnv }
func (s *PlanEnv) Get(sym string) exp.Resolver {
	// resolve from added tasks
	if sym[0] != '/' {
		return nil
	}
	if len(sym) == 1 {
		return exp.LitResolver{s.Result}
	}
	path, err := lit.ReadPath(sym[1:])
	if err != nil {
		return nil
	}
	sym = path[0].Key
	for _, t := range s.Root {
		if t.Name == sym {
			return &TaskResolver{t, path[1:]}
		}
	}
	return nil
}

type TaskEnv struct {
	Par exp.Env
	*Task
	Param lit.Lit
}

func (s *TaskEnv) Parent() exp.Env      { return s.Par }
func (s *TaskEnv) Supports(x byte) bool { return x == '.' }

func (s *TaskEnv) Def(sym string, r exp.Resolver) error { return exp.ErrNoDefEnv }
func (s *TaskEnv) Get(sym string) exp.Resolver {
	if sym[0] != '.' {
		return nil
	}
	sym = sym[1:]
	if s.Query != nil {
		for _, t := range s.Query.Sel {
			if t.Name == sym {
				return &TaskResolver{t, nil}
			}
		}
		if s.Param != nil {
			l, err := lit.Select(s.Param, sym)
			if err == nil {
				return exp.LitResolver{l}
			}
		} else {
			// otherwise check query result type
			p, _, err := s.Query.Type.ParamByKey(sym)
			if err == nil {
				return exp.TypedUnresolver{p.Type}
			}
		}
	}
	// resolves to result from result type
	p, _, err := s.Type.ParamByKey(sym)
	if err == nil {
		return exp.TypedUnresolver{p.Type}
	}
	return nil
}

func FindEnv(env exp.Env) *PlanEnv {
	for env != nil {
		env = exp.Supports(env, '/')
		if p, ok := env.(*PlanEnv); ok {
			return p
		}
		if env != nil {
			env = env.Parent()
		}
	}
	return nil
}

type TaskResolver struct {
	*Task
	Rest lit.Path
}

func (r *TaskResolver) Resolve(c *exp.Ctx, env exp.Env, e exp.El, hint typ.Type) (exp.El, error) {
	if r.Done {
		if len(r.Rest) != 0 {
			return lit.SelectPath(r.Result, r.Rest)
		}
		return r.Result, nil
	}
	if e.Typ().Kind == typ.ExpSym {
		t := r.Type
		if len(r.Rest) != 0 {
			l, err := lit.SelectPath(t, r.Rest)
			if err != nil {
				return nil, err
			}
			t = l.(typ.Type)
		}
		e.(*exp.Sym).Type = t
	}
	return e, exp.ErrUnres
}
