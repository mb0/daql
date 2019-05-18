package qry

import (
	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/std"
	"github.com/mb0/xelf/typ"
	"github.com/mb0/xelf/utl"
)

var Builtin = exp.Builtin{
	utl.StrLib.Lookup(),
	utl.TimeLib.Lookup(),
	std.Core, std.Decl,
}

func NewEnv(env exp.Env, pr *dom.Project, bend Backend) *PlanEnv {
	if env == nil {
		env = Builtin
	}
	domEnv := dom.NewEnv(env, pr)
	s := &exp.ParamScope{exp.NewScope(domEnv), nil}
	return &PlanEnv{s, pr, &Plan{}, nil, bend}
}

type PlanEnv struct {
	Par     exp.Env
	Project *dom.Project
	*Plan
	*Result
	Backend
}

func (s *PlanEnv) Parent() exp.Env      { return s.Par }
func (s *PlanEnv) Supports(x byte) bool { return x == '/' }
func (s *PlanEnv) Get(sym string) *exp.Def {
	if sym == "qry" {
		return exp.NewDef(qrySpec)
	}
	// resolve from added tasks
	if sym[0] != '/' {
		return nil
	}
	if len(sym) == 1 {
		return &exp.Def{Type: s.Type}
	}
	path, err := lit.ReadPath(sym[1:])
	if err != nil {
		return nil
	}
	sym = path[0].Key
	for _, t := range s.Root {
		if t.Name == sym {
			return taskDef(t, path[1:], s)
		}
	}
	return nil
}

type TaskEnv struct {
	Par  exp.Env
	Penv *PlanEnv
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
				return taskDef(t, nil, s.Penv)
			}
		}
		if s.Param != nil {
			l, err := lit.Select(s.Param, sym)
			if err == nil {
				return exp.NewDef(l)
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

func taskDef(t *Task, path lit.Path, r *PlanEnv) *exp.Def {
	if r.Result != nil {
		nfo := r.Info[t]
		if nfo.Done {
			l, err := lit.SelectPath(nfo.Data, path)
			if err != nil {
				return nil
			}
			return exp.NewDef(l)
		}
	}
	l, err := lit.SelectPath(t.Type, path)
	if err != nil {
		return nil
	}
	return &exp.Def{Type: l.(typ.Type)}
}
