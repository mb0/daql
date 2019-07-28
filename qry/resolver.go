package qry

import (
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/std"
	"github.com/mb0/xelf/typ"
)

var qrySpec = exp.ImplementReq("(form 'qry' :args? :decls? : @1)", false,
	func(x exp.ReslReq) (exp.El, error) {
		qenv := FindEnv(x.Env)
		if qenv == nil {
			return nil, cor.Errorf("no qry environment for query %s", x)
		}
		p := &Doc{}
		args := x.Args(0)
		penv := &DocEnv{x.Env, p}
		if len(args) > 0 {
			// simple query
			if len(x.Args(1)) > 0 {
				return nil, cor.Errorf("either use simple or compound query got %v rest %v",
					args, x.Args(1))
			}
			t, err := resolveTask(x.Ctx, penv, exp.NewNamed("", args...), nil)
			if err != nil {
				return nil, err
			}
			p.Root = []*Task{t}
			p.Type = t.Type
		} else {
			decls, err := x.Decls(1)
			if err != nil {
				return nil, err
			}
			ps := make([]typ.Param, 0, len(decls))
			for _, d := range decls {
				t, err := resolveTask(x.Ctx, penv, d, nil)
				if err != nil {
					return nil, err
				}
				p.Root = append(p.Root, t)
				ps = append(ps, typ.Param{Name: t.Name, Type: t.Type})
			}
			p.Type = typ.Rec(ps)
		}
		if len(p.Root) == 0 {
			return nil, cor.Error("empty plan")
		}
		return &exp.Atom{Lit: &exp.Spec{typ.Func("", []typ.Param{
			{"arg", typ.Any},
			{"", p.Type},
		}), p}}, nil
	})

var taskSig = exp.MustSig("(form '_' :ref? @1 :args? :decls? : void)")

func resolveTask(c *exp.Ctx, env exp.Env, d *exp.Named, p *Task) (t *Task, err error) {
	t = &Task{Parent: p}
	if d.Name != "" {
		t.Name = d.Name[1:]
	}
	var fst exp.El
	if d.El == nil {
		// must be field selection in a parent query
		// this transforms +id to (+id .id) an + to (+ .)
		fst = &exp.Sym{Name: "." + t.Name}
	} else {
		lo, err := exp.LayoutArgs(taskSig, d.Args())
		if err != nil {
			return nil, err
		}
		fst = lo.Arg(0)
		switch sym := fst.String(); sym[0] {
		case '?', '*', '#':
			if d.Name == "+" {
				t.Name = cor.LastKey(sym)
			}
			err = resolveQuery(c, env, t, sym, lo)
			if err != nil {
				return nil, err
			}
			return t, nil
		case '.':
		default:
			fst = &exp.Dyn{Els: d.Args()}
		}
	}
	if t.Name == "" {
		return nil, cor.Errorf("unnamed expr task %s", d)
	}
	// partially resolve expression
	fst, err = exp.Resolve(env, fst)
	if fst == nil {
		return nil, cor.Errorf("resolve task %s: %v", d, err)
	}
	t.Expr = fst
	if err == nil {
		t.Type = fst.Typ()
		return t, nil
	} else if err == exp.ErrUnres {
		// check for sym, form or func expression to find a result type
		var rt typ.Type
		switch v := fst.(type) {
		case *exp.Sym:
			rt = v.Type
		case *exp.Call:
			if v.Spec != nil {
				rt = v.Type.Params[len(v.Type.Params)-1].Type
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

func resolveQuery(c *exp.Ctx, env exp.Env, t *Task, ref string, lo *exp.Layout) error {
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
	err := resolveTag(c, tenv, q, args)
	if err != nil {
		return err
	}
	// TODO check that order only accesses result fields
	rt := scalar
	if rt == typ.Void {
		sel := lo.Args(2)
		rt, err = resolveSel(c, tenv, q, sel)
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

func resolveTag(c *exp.Ctx, env exp.Env, q *Query, args []exp.El) (err error) {
	if q.Whr == nil {
		q.Whr = &exp.Dyn{}
	}
	for _, arg := range args {
		tag, ok := arg.(*exp.Named)
		if !ok {
			q.Whr.Els = append(q.Whr.Els, arg)
			continue
		}
		switch tag.Name {
		case ":whr":
			q.Whr.Els = append(q.Whr.Els, tag.Args()...)
		case ":lim":
			// takes one number
			q.Lim, err = resolveInt(c, env, tag.Args())
		case ":off":
			// takes one or two number or a list of two numbers
			q.Off, err = resolveInt(c, env, tag.Args())
		case ":ord", ":asc", ":desc":
			// takes one or more field references
			// can be used multiple times to append to order
			err = resolveOrd(c, env, q, tag.Name == ":desc", tag.Args())
		default:
			return cor.Errorf("unexpected query tag %q", tag.Name)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func resolveInt(c *exp.Ctx, env exp.Env, args []exp.El) (int64, error) {
	el, err := c.Resolve(env, &exp.Dyn{Els: args}, typ.Int)
	if err != nil {
		return 0, err
	}
	n, ok := el.(*exp.Atom).Lit.(lit.Numeric)
	if !ok {
		return 0, cor.Errorf("expect int got %s", el.Typ())
	}
	return int64(n.Num()), nil
}

func resolveOrd(c *exp.Ctx, env exp.Env, q *Query, desc bool, args []exp.El) error {
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

func resolveSel(c *exp.Ctx, env *SelEnv, q *Query, args []exp.El) (typ.Type, error) {
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
					t, err := resolveTask(c, env, d, env.Task)
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
