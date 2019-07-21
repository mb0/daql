package genpg

import (
	"fmt"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

type ParamEnv struct {
	Env
	Params []Param
}

type Param struct {
	Name string
	Type typ.Type
}

func (pe *ParamEnv) Translate(s *exp.Sym) (string, lit.Lit, error) {
	for i, p := range pe.Params {
		if p.Name == s.Name {
			return fmt.Sprintf("$%d", i+1), nil, nil
		}
	}
	if pe.Env == nil {
		return "", nil, exp.ErrUnres
	}
	n, l, err := pe.Env.Translate(s)
	if err == External {
		if n == "" {
			n = s.Name
		}
		pe.Params = append(pe.Params, Param{n, s.Type})
		return fmt.Sprintf("$%d", len(pe.Params)), nil, nil
	}
	return n, l, err
}

type ExpEnv struct {
	exp.Env
}

func (ee ExpEnv) Translate(s *exp.Sym) (string, lit.Lit, error) {
	d := ee.Get(s.Name)
	if d == nil {
		return "", nil, exp.ErrUnres
	}
	if d.Lit != nil {
		return "", d.Lit, nil
	}
	n := s.Name
	if n[0] == '.' {
		n = n[1:]
	}
	if cor.IsKey(n) {
		return n, nil, nil
	}
	return s.Name, nil, External
}
