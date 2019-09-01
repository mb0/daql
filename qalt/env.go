package qry

import (
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

type QryEnv struct {
	Par exp.Env
	Planner
}

func NewEnv(env exp.Env, bends ...Backend) *QryEnv {
	return &QryEnv{Par: env, Planner: &planner{Backends: bends}}
}

func (qe *QryEnv) Parent() exp.Env      { return qe.Par }
func (qe *QryEnv) Supports(x byte) bool { return x == '?' }
func (qe *QryEnv) Get(sym string) *exp.Def {
	switch sym[0] {
	case '?', '*', '#':
		query := qe.Prep(sym)
		spec := &exp.Spec{typ.Form(sym, qrySig.Params), query}
		return exp.NewDef(spec)
	}
	return nil
}

func (qe *QryEnv) Qry(q string, arg lit.Lit) (lit.Lit, error) {
	el, err := exp.Read(strings.NewReader(q))
	if err != nil {
		return nil, cor.Errorf("read qry %s error: %w", q, err)
	}
	if arg == nil {
		arg = lit.Nil
	}
	// TODO use param scope with arg
	for i := 0; i < 255; i++ {
		l, err := exp.Eval(qe, el)
		if err != nil {
			if err == exp.ErrUnres {
				el = l
				continue
			}
			return nil, cor.Errorf("eval qry %s error: %w", el, err)
		}
		el = l
	}
	if a, ok := el.(*exp.Atom); ok {
		return a.Lit, nil
	}
	return nil, cor.Errorf("qry result %T is not an atom", el)
}
