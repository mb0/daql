package qry

import (
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/std"
	"github.com/mb0/xelf/typ"
)

var qrySpec = std.SpecXX("(form 'qry' :args? :decls? : @1)", func(x std.CallCtx) (exp.El, error) {
	qenv := FindEnv(x.Env)
	if qenv == nil {
		return nil, cor.Errorf("no qry environment for query %s", x)
	}
	doc := &Doc{}
	args := x.Args(0)
	penv := exp.NewParamReslEnv(x.Env, x.Ctx)
	denv := doc.ReslEnv(penv)
	if len(args) > 0 {
		// simple query
		if len(x.Args(1)) > 0 {
			return nil, cor.Errorf("either use simple or compound query got %v rest %v",
				args, x.Args(1))
		}
		t, err := resolveTask(x.Prog, denv, exp.NewNamed("", args...), nil)
		if err != nil {
			return nil, err
		}
		doc.Root = []*Task{t}
		doc.Type = t.Type
	} else {
		decls, err := x.Decls(1)
		if err != nil {
			return nil, err
		}
		ps := make([]typ.Param, 0, len(decls))
		for _, d := range decls {
			t, err := resolveTask(x.Prog, denv, d, nil)
			if err != nil {
				return nil, err
			}
			doc.Root = append(doc.Root, t)
			ps = append(ps, typ.Param{Name: t.Name, Type: t.Type})
		}
		doc.Type = typ.Rec(ps)
	}
	if len(doc.Root) == 0 {
		return nil, cor.Error("empty plan")
	}
	return &exp.Atom{Lit: &exp.Spec{typ.Func("", []typ.Param{
		{"arg", typ.Any},
		{"", doc.Type},
	}), doc}}, nil
})

var taskSig = exp.MustSig("(form '_' :ref? @1 :args? :decls? : void)")

func resolveTask(p *exp.Prog, env exp.Env, d *exp.Named, par *Task) (t *Task, err error) {
	t = &Task{Parent: par}
	if d.Name != "" {
		t.Name = d.Name[1:]
	}
	var fst exp.El
	if d.El == nil {
		// must be field selection in a parent query
		// this transforms +id to (+id .id) an + to (+ .)
		fst = &exp.Sym{Name: "." + t.Name}
	} else {
		lo, err := exp.FormLayout(taskSig, d.Args())
		if err != nil {
			return nil, err
		}
		fst = lo.Arg(0)
		switch sym := fst.String(); sym[0] {
		case '?', '*', '#':
			if d.Name == "+" {
				t.Name = cor.LastKey(sym)
			}
			err = resolveQuery(p, env, t, sym, lo)
			if err != nil {
				return nil, err
			}
			return t, nil
		default:
			fst = &exp.Dyn{Els: d.Args()}
		}
	}
	if t.Name == "" {
		return nil, cor.Errorf("unnamed expr task %s", d)
	}
	// partially resolve expression
	fst, err = exp.Resl(env, fst)
	if fst == nil {
		return nil, cor.Errorf("resolve task %s: %v", d, err)
	}
	t.Expr = fst
	if err == nil {
		t.Type = exp.ResType(fst)
		return t, nil
	} else if err == exp.ErrUnres {
		// check for sym, form or func expression to find a result type
		var rt typ.Type
		switch v := fst.(type) {
		case *exp.Sym:
			rt = v.Type
		case *exp.Call:
			if v.Spec != nil {
				rt = v.Sig.Params[len(v.Sig.Params)-1].Type
			}
		}
		switch rt.Kind {
		case typ.KindRef, typ.KindVoid:
		default:
			t.Type = rt
		}
	}
	if t.Type == typ.Void {
		return nil, cor.Errorf("no type for task %s, %s in %T", d, fst, env)
	}
	// this is it, we handle the final resolution after planning
	return t, nil
}

var andSpec = std.Core("and")

func resolveQuery(p *exp.Prog, env exp.Env, t *Task, ref string, lo *exp.Layout) error {
	q := &Query{Ref: ref}
	name := ref[1:]
	if name == "" {
		return cor.Error("empty query reference")
	}
	// locate the plan environment for a project and find the model
	penv := FindEnv(env)
	scalar := typ.Void
	switch name[0] {
	case '.', '/', '$': // path
		return cor.Error("path query reference not yet implemented")
	default:
		// lookup schema
		s := strings.SplitN(name, ".", 3)
		if len(s) < 2 {
			return cor.Errorf("unknown schema name %q", ref)
		}
		if penv == nil || penv.Project == nil {
			return cor.Errorf("no project found in plan env %v", penv)
		}
		m := penv.Project.Schema(s[0]).Model(s[1])
		if m != nil {
			q.Type = m.Type
		}
		if len(s) > 2 {
			f := m.Field(s[2])
			if f.Param == nil {
				return cor.Errorf("no field found for %q", ref)
			}
			scalar = f.Type
		}
	}
	// at this point we need to have the result type to inform argument parsing
	if q.Type == typ.Void {
		return cor.Errorf("no type found for %q", ref)
	}
	t.Query = q
	args := lo.Args(1)
	tenv := &SelEnv{env, t}
	err := resolveTag(p, tenv, q, args)
	if err != nil {
		return err
	}
	// TODO check that order only accesses result fields
	rt := scalar
	if rt == typ.Void {
		sel := lo.Args(2)
		rt, err = resolveSel(p, tenv, q, sel)
		if err != nil {
			return err
		}
	}
	// set the task result type based on the query subject type
	switch ref[0] {
	case '?':
		t.Type = typ.Opt(rt)
		if q.Lim > 1 {
			return cor.Errorf("unexpected limit %d for single result", q.Lim)
		}
		q.Lim = 1
	case '*':
		t.Type = typ.List(rt)
	case '#':
		t.Type = typ.Int
	}
	return nil
}

func resolveTag(p *exp.Prog, env exp.Env, q *Query, args []exp.El) (err error) {
	var whr []exp.El
	for _, arg := range args {
		tag, ok := arg.(*exp.Named)
		if !ok {
			whr = append(whr, arg)
			continue
		}
		switch tag.Name {
		case ":whr":
			whr = append(whr, tag.Args()...)
		case ":lim":
			// takes one number
			q.Lim, err = resolveInt(p, env, tag.Args())
		case ":off":
			// takes one or two number or a list of two numbers
			q.Off, err = resolveInt(p, env, tag.Args())
		case ":ord", ":asc", ":desc":
			// takes one or more field references
			// can be used multiple times to append to order
			err = resolveOrd(p, env, q, tag.Name == ":desc", tag.Args())
		default:
			return cor.Errorf("unexpected query tag %q", tag.Name)
		}
		if err != nil {
			return err
		}
	}
	if len(whr) > 0 {
		q.Whr = &exp.Dyn{Els: whr}
	}
	return nil
}

func resolveInt(p *exp.Prog, env exp.Env, args []exp.El) (int64, error) {
	el, err := p.Eval(env, &exp.Dyn{Els: args}, typ.Int)
	if err != nil {
		return 0, err
	}
	n, ok := el.(*exp.Atom).Lit.(lit.Numeric)
	if !ok {
		return 0, cor.Errorf("expect int got %s", el.Typ())
	}
	return int64(n.Num()), nil
}

func resolveOrd(p *exp.Prog, env exp.Env, q *Query, desc bool, args []exp.El) error {
	// either takes a list of strings, one string or one local symbol
	for _, arg := range args {
		sym, ok := arg.(*exp.Sym)
		if !ok {
			return cor.Errorf("want order symbol got %T", arg)
		}
		q.Ord = append(q.Ord, Ord{"." + sym.Key(), desc})
	}
	return nil
}

func resolveSel(p *exp.Prog, env *SelEnv, q *Query, args []exp.El) (typ.Type, error) {
	var ps []typ.Param
	if q.Type.Kind&typ.MaskElem == typ.KindRec && q.Type.HasParams() {
		ps = q.Type.Params
	}
	// start with all fields unless we start with "-"
	res := make([]*Task, 0, len(ps)+len(args))
	for _, p := range ps {
		res = append(res, &Task{Name: p.Name, Type: p.Type, Parent: env.Task})
	}
	if len(args) == 0 {
		q.Sel = res
		return q.Type, nil
	}
	for i, arg := range args {
		d, ok := arg.(*exp.Named)
		if !ok || d.Name == "" {
			return typ.Void, cor.Errorf("expect declaration got %T", arg)
		}
		args := d.Args()
		switch d.Name[0] {
		case '-':
			if len(d.Name) == 1 {
				if len(args) > 0 {
					// remove all arguments from fields
					keys, err := getKeys(args)
					if err != nil {
						return typ.Void, err
					}
					for _, key := range keys {
						res, err = removeKey(res, key)
						if err != nil {
							return typ.Void, err
						}
					}
				} else if i == 0 {
					// else start without
					res = res[:0]
				}
			} else {
				if len(args) > 0 {
					return typ.Void, cor.Errorf("unexpected arguments %s", d)
				}
				// remove from fields
				var err error
				res, err = removeKey(res, strings.ToLower(d.Name[1:]))
				if err != nil {
					return typ.Void, err
				}
			}
		case '+':
			if len(d.Name) == 1 {
				if len(args) > 0 {
				} else {
					if i == 0 { // reset with explicit decls
						res = res[:0]
					}
					// include all arguments as fields
					keys, err := getKeys(args)
					if err != nil {
						return typ.Void, err
					}
					add, err := getParams(ps, keys...)
					if err != nil {
						return typ.Void, err
					}
					res, err = addParams(res, add, env.Task)
					if err != nil {
						return typ.Void, err
					}
				}
			} else {
				if len(args) > 0 {
					// add arguments as task to sel
					t, err := resolveTask(p, env, d, env.Task)
					if err != nil {
						return typ.Void, err
					}
					res = append(res, t)
				} else {
					if i == 0 { // reset with explicit decls
						res = res[:0]
					}
					add, err := getParams(ps, strings.ToLower(d.Name[1:]))
					if err != nil {
						return typ.Void, err
					}
					res, err = addParams(res, add, env.Task)
					if err != nil {
						return typ.Void, err
					}
				}
			}
		}
	}
	ps = make([]typ.Param, 0, len(res))
	for _, t := range res {
		ps = append(ps, typ.Param{Name: t.Name, Type: t.Type})
	}
	q.Sel = res
	return typ.Rec(ps), nil
}

func getKeys(args []exp.El) ([]string, error) {
	res := make([]string, 0, len(args))
	for _, e := range args {
		s, ok := e.(*exp.Sym)
		if !ok {
			return nil, cor.Errorf("want sym got %T", e)
		}
		key := strings.TrimPrefix(strings.ToLower(s.Name), ".")
		res = append(res, key)
	}
	return res, nil
}

func getParams(ps []typ.Param, keys ...string) ([]typ.Param, error) {
	res := make([]typ.Param, 0, len(keys))
NextKey:
	for _, key := range keys {
		for _, p := range ps {
			if p.Key() == key {
				res = append(res, p)
				continue NextKey
			}
		}
		return nil, cor.Errorf("key %q not part of subject %v", key, ps)
	}
	return res, nil
}

func addParams(res []*Task, ps []typ.Param, pt *Task) ([]*Task, error) {
	for _, p := range ps {
		for _, t := range res {
			if strings.EqualFold(t.Name, p.Name) {
				return nil, cor.Errorf("key %q already in result %v", p.Key(), res)
			}
		}
		res = append(res, &Task{Name: p.Name, Type: p.Type, Parent: pt})
	}
	return res, nil
}

func removeKey(res []*Task, key string) ([]*Task, error) {
	for i, t := range res {
		if strings.EqualFold(t.Name, key) {
			return append(res[:i], res[i+1:]...), nil
		}
	}
	return nil, cor.Errorf("key %q not found in %v", key, res)
}
