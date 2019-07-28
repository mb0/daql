package genpg

import (
	"fmt"

	"github.com/mb0/daql/gen"
	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

type Writer struct {
	gen.Gen
	Translator
	Params []Param
}

type Param struct {
	Name  string
	Type  typ.Type
	Value lit.Lit
}

func NewWriter(b bfr.B, t Translator) *Writer {
	return &Writer{gen.Gen{
		Ctx:    bfr.Ctx{B: b, Tab: "\t"},
		Header: "-- generated code\n\n",
	}, t, nil}
}
func (w *Writer) Translate(env exp.Env, s *exp.Sym) (string, lit.Lit, error) {
	for i, p := range w.Params {
		// TODO better way to idetify a reference, maybe in another env
		if p.Name == s.Name {
			return fmt.Sprintf("$%d", i+1), nil, nil
		}
	}
	if w.Translator == nil {
		return "", nil, exp.ErrUnres
	}
	n, l, err := w.Translator.Translate(env, s)
	if err == External {
		if n == "" {
			n = s.Name
		}
		w.Params = append(w.Params, Param{n, s.Type, l})
		return fmt.Sprintf("$%d", len(w.Params)), nil, nil
	}
	return n, l, err
}

type Translator interface {
	Translate(exp.Env, *exp.Sym) (string, lit.Lit, error)
}

type ExpEnv struct{}

func (ee ExpEnv) Translate(env exp.Env, s *exp.Sym) (string, lit.Lit, error) {
	d := env.Get(s.Name)
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
	return s.Name, d.Lit, External
}
