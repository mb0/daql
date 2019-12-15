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

type Ord struct {
	Key  string
	Desc bool
	Subj bool
}

// Query describes a query definition.
type Query struct {
	Kind Kind
	Ref  string
	Subj typ.Type

	Sel *Sel
	Res typ.Type
	Whr []exp.El
	Lim int64
	Off int64
	Ord []Ord

	Planner Planner
	Model   *dom.Model
	Bend    Backend
	Err     error
}

func splitPlain(args []exp.El) (plain, rest []exp.El) {
	for i, arg := range args {
		switch t := arg.(type) {
		case *exp.Tag:
			return args[:i], args[i:]
		case *exp.Sym:
			switch t.Name {
			case "_", "+", "-":
				return args[:i], args[i:]
			}
		}
	}
	return args, nil
}
func splitDecls(args []exp.El) (tags, decl []*exp.Tag) {
	tags = make([]*exp.Tag, 0, len(args))
	decl = make([]*exp.Tag, 0, len(args))
	for _, arg := range args {
		switch t := arg.(type) {
		case *exp.Tag:
			if len(decl) == 0 && cor.IsKey(t.Name) && t.Name[0] != '_' {
				tags = append(tags, t)
			} else {
				decl = append(decl, t)
			}
		case *exp.Sym:
			decl = append(decl, &exp.Tag{Name: t.Name, Src: t.Src})
		default:
			decl = append(decl, &exp.Tag{El: arg, Src: arg.Source()})
		}
	}
	return tags, decl
}

var qrySig = exp.MustSig("<form qry tail?; @>")

func (q *Query) Resl(p *exp.Prog, env exp.Env, c *exp.Call, h typ.Type) (exp.El, error) {
	renv, ok := exp.Supports(env, '?').(*ReslEnv)
	if ok {
		renv.AddNested(q)
	}
	if q.Err != nil {
		return c, q.Err
	}
	if q.Subj == typ.Void {
		if q.Model != nil {
			q.Subj = q.Model.Type
		} else {
			ref := &exp.Sym{Name: q.Ref}
			_, err := p.Resl(env, ref, typ.Void)
			if err != nil {
				return c, err
			}
			q.Subj = ref.Type
		}
	}
	whr, args := splitPlain(c.Args(0))
	tags, decl := splitDecls(args)
	if q.Sel == nil {
		// resolve selection
		sel, err := reslSel(p, env, q, decl)
		if err != nil {
			return c, cor.Errorf("resl sel %v: %v", decl, err)
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
	for _, tag := range tags {
		var err error
		switch tag.Name {
		case "whr":
			whr = append(whr, tag.Args()...)
		case "lim":
			q.Lim, err = evalInt(p, env, tag.El)
		case "off":
			q.Off, err = evalInt(p, env, tag.El)
		case "ord", "asc", "desc":
			// takes one or more field references
			// can be used multiple times to append to order
			err = evalOrd(p, env, q, tag.Name == "desc", tag.Args())
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
		if !ok || sym.Name == "" {
			return cor.Errorf("order want sym got %s", arg)
		}
		key := sym.Name
		var subj bool
		_, _, err := q.Sel.Type.ParamByKey(key)
		if err != nil {
			subj = true
			_, _, err = q.Subj.ParamByKey(key)
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
