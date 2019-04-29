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

func QryLookup(sym string) *exp.Spec {
	if sym == "qry" {
		return qrySpec
	}
	return nil
}

type PlanEnv struct {
	Par exp.Env
	*Plan
	Backend
}

func (p *PlanEnv) Parent() exp.Env    { return p.Par }
func (*PlanEnv) Supports(x byte) bool { return x == '/' }
func (s *PlanEnv) Get(sym string) *exp.Def {
	// resolve from added tasks
	if sym[0] != '/' {
		return nil
	}
	if len(sym) == 1 {
		return exp.DefLit(s.Result)
	}
	path, err := lit.ReadPath(sym[1:])
	if err != nil {
		return nil
	}
	sym = path[0].Key
	for _, t := range s.Root {
		if t.Name == sym {
			return taskDef(t, path[1:])
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

func (s *TaskEnv) Get(sym string) *exp.Def {
	if sym[0] != '.' {
		return nil
	}
	sym = sym[1:]
	if s.Query != nil {
		for _, t := range s.Query.Sel {
			if t.Name == sym {
				return taskDef(t, nil)
			}
		}
		if s.Param != nil {
			l, err := lit.Select(s.Param, sym)
			if err == nil {
				return exp.DefLit(l)
			}
		} else {
			// otherwise check query result type
			p, _, err := s.Query.Type.ParamByKey(sym)
			if err == nil {
				return &exp.Def{Type: p.Type}
			}
		}
	}
	// resolves to result from result type
	p, _, err := s.Type.ParamByKey(sym)
	if err == nil {
		return &exp.Def{Type: p.Type}
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

func taskDef(t *Task, path lit.Path) *exp.Def {
	if t.Done {
		if len(path) != 0 {
			l, err := lit.SelectPath(t.Result, path)
			if err != nil {
				return nil
			}
			return exp.DefLit(l)
		}
		return exp.DefLit(t.Result)
	}
	if len(path) != 0 {
		l, err := lit.SelectPath(t.Type, path)
		if err != nil {
			return nil
		}
		return &exp.Def{Type: l.(typ.Type)}
	}
	return &exp.Def{Type: t.Type}
}
