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

// QryEnv provide the qry form and the required facilities for resolving and executing queries.
type QryEnv struct {
	Par     exp.Env
	Project *dom.Project
	*Plan
	*Result
	Backend
}

func NewEnv(env exp.Env, pr *dom.Project, bend Backend) *QryEnv {
	if env == nil {
		env = Builtin
	}
	domEnv := dom.NewEnv(env, pr)
	s := &exp.ParamScope{exp.NewScope(domEnv), nil}
	return &QryEnv{s, pr, &Plan{}, nil, bend}
}

func FindEnv(env exp.Env) *QryEnv {
	for env != nil {
		env = exp.Supports(env, '?')
		if p, ok := env.(*QryEnv); ok {
			return p
		}
		if env != nil {
			env = env.Parent()
		}
	}
	return nil
}

func (qe *QryEnv) Parent() exp.Env      { return qe.Par }
func (qe *QryEnv) Supports(x byte) bool { return x == '?' }
func (qe *QryEnv) Get(sym string) *exp.Def {
	if sym == "qry" {
		return exp.NewDef(qrySpec)
	}
	return nil
}

type PlanEnv struct {
	Par exp.Env
	*QryEnv
}

func (pe *PlanEnv) Parent() exp.Env      { return pe.Par }
func (pe *PlanEnv) Supports(x byte) bool { return x == '/' }
func (pe *PlanEnv) Get(sym string) *exp.Def {
	if sym[0] != '/' {
		return nil
	}
	if len(sym) == 1 {
		return &exp.Def{Type: pe.Type}
	}
	path, err := lit.ReadPath(sym[1:])
	if err != nil {
		return nil
	}
	sym = path[0].Key
	for _, t := range pe.Root {
		if t.Name != sym {
			continue
		}
		l, err := lit.SelectPath(t.Type, path[1:])
		if err != nil {
			return nil
		}
		return &exp.Def{Type: l.(typ.Type)}
	}
	return nil
}

type ExecEnv struct {
	Par exp.Env
	*QryEnv
}

func (ee *ExecEnv) Parent() exp.Env      { return ee.Par }
func (ee *ExecEnv) Supports(x byte) bool { return x == '/' }
func (ee *ExecEnv) Get(sym string) *exp.Def {
	if sym[0] != '/' {
		return nil
	}
	if len(sym) == 1 {
		return exp.NewDef(ee.Data)
	}
	path, err := lit.ReadPath(sym[1:])
	if err != nil {
		return nil
	}
	sym = path[0].Key
	for _, t := range ee.Root {
		if t.Name != sym {
			continue
		}
		nfo := ee.Info[t]
		if nfo.Done {
			l, err := lit.SelectPath(nfo.Data, path[1:])
			if err != nil {
				return nil
			}
			return exp.NewDef(l)
		}
	}
	return nil
}

type TaskEnv struct {
	Par  exp.Env
	Penv *QryEnv
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
			if t.Name != sym {
				continue
			}
			if s.Penv.Result != nil {
				nfo := s.Penv.Info[t]
				if nfo.Done {
					return exp.NewDef(nfo.Data)
				}
			}
			return &exp.Def{Type: t.Type}
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
