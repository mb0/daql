// Package qry provide a way to work with external data, but can be used for local data as well.
// Backends provide model definitions and a way to evaluate queries to external data.
// The query environment detects query subjects and provides the planner for a program that manages
// query evaluation.
// Programs with queries take at least two evaluation passes. Each pass the resolved queries are
// planned as jobs for execution and all jobs from previous iterations are executed.
//
// TODO: think about whether or how to support queries in loops
package qry

import (
	"log"

	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

type Kind byte

const (
	KindOne   Kind = '?'
	KindMany  Kind = '*'
	KindCount Kind = '#'
)

type Subj struct {
	Kind  Kind
	Ref   string
	Path  string
	Model *dom.Model
	Bend  Backend
	Type  typ.Type
}

type Ord struct {
	Key  string
	Desc bool
	Subj bool
}

// Query describes a query definition.
type Query struct {
	Planner Planner
	Subj
	Sel *Sel
	Res typ.Type
	Whr []exp.El
	Lim int64
	Off int64
	Ord []Ord

	Err error
}

var qrySig = exp.MustSig("(form 'qry' :args? :decls? : @)")

func (q *Query) Resl(p *exp.Prog, env exp.Env, c *exp.Call, h typ.Type) (exp.El, error) {
	log.Printf("resl %s", c)
	if q.Err != nil {
		return c, q.Err
	}
	if q.Subj.Type == typ.Void {
		if q.Model != nil {
			q.Subj.Type = q.Model.Type
		} else {
			ref := &exp.Sym{Name: q.Ref}
			_, err := p.Resl(env, ref, typ.Void)
			if err != nil {
				return c, err
			}
			q.Subj.Type = ref.Type
		}
	}
	if q.Sel == nil {
		// resolve selection
		decls, err := c.Decls(1)
		if err != nil {
			return c, err
		}
		if q.Path != "" {
			if len(decls) == 0 {
				decls = append(decls, &exp.Named{Name: "-"})
			}
			decls = append(decls, &exp.Named{Name: "+" + q.Path})
		}
		sel, err := reslSel(p, env, q.Subj.Type, decls)
		if err != nil {
			return c, cor.Errorf("resl sel %v: %v", decls, err)
		}
		q.Sel = sel
	}
	if q.Res == typ.Void {
		// now check the query kind an selection to arrive at the result type
		res, err := resType(q, q.Sel.Type)
		if err != nil {
			return c, cor.Errorf("resl res %v: %v", err)
		}
		q.Res = res
	}
	// resolve arguments for whr ord lim and off
	var whr []exp.El
	for _, arg := range c.Args(0) {
		tag, ok := arg.(*exp.Named)
		if !ok {
			whr = append(whr, arg)
			continue
		}
		var err error
		switch tag.Name {
		case ":whr":
			whr = append(whr, tag.Args()...)
		case ":lim":
			q.Lim, err = evalInt(p, env, tag.El)
		case ":off":
			q.Off, err = evalInt(p, env, tag.El)
		case ":ord", ":asc", ":desc":
			// takes one or more field references
			// can be used multiple times to append to order
			err = evalOrd(p, env, q, tag.Name == ":desc", tag.Args())
		default:
			return c, cor.Errorf("unexpected query tag %q", tag.Name)
		}
		if err != nil {
			return c, err
		}
	}
	q.Whr = whr
	return c, nil
}

func (q *Query) Eval(p *exp.Prog, env exp.Env, c *exp.Call, h typ.Type) (exp.El, error) {
	if q.Sel == nil {
		_, err := q.Resl(p, env, c, h)
		if err != nil {
			return nil, err
		}
	}
	// query arguments must all be evaluated or else skip until next run
	if q.Model == nil { // exec local query
		l, err := execLocalQuery(p, &Job{Query: q, Env: env})
		if err != nil {
			return c, err
		}
		return &exp.Atom{Lit: l, Src: c.Src}, nil
	}
	return q.Planner.Plan(p, env, c, q)
}

func resType(q *Query, sel typ.Type) (typ.Type, error) {
	if q.Kind == KindCount {
		return typ.Int, nil
	}
	if q.Model != nil && q.Path != "" {
		// scalar selection from sel type
		st, err := typ.Select(sel, q.Path)
		if err != nil {
			return typ.Void, err
		}
		sel = st
	}
	switch q.Kind {
	case KindOne:
		return typ.Opt(sel), nil
	case KindMany:
		return typ.List(sel), nil
	}
	return typ.Void, cor.Errorf("unexpected query kind %q", q.Kind)
}

func evalOrd(p *exp.Prog, env exp.Env, q *Query, desc bool, args []exp.El) error {
	for _, arg := range args {
		sym, ok := arg.(*exp.Sym)
		if !ok || sym.Name == "" || sym.Name[0] != '.' {
			return cor.Errorf("order want local sym got %s", arg)
		}
		key := sym.Name[1:]
		var subj bool
		_, _, err := q.Sel.Type.ParamByKey(key)
		if err != nil {
			subj = true
			_, _, err = q.Subj.Type.ParamByKey(key)
			if err != nil {
				return cor.Errorf("order sym %q not found", sym.Name)
			}
		}
		q.Ord = append(q.Ord, Ord{key, desc, subj})
	}
	return nil
}

func evalInt(p *exp.Prog, env exp.Env, arg exp.El) (int64, error) {
	el, err := p.Eval(env, arg, typ.Int)
	if err != nil {
		return 0, err
	}
	n, ok := el.(*exp.Atom).Lit.(lit.Numeric)
	if !ok {
		return 0, cor.Errorf("expect int got %s", el.Typ())
	}
	return int64(n.Num()), nil
}
