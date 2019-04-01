package qry

import (
	"strings"

	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

var qryForm *exp.Form

func init() {
	qryForm = &exp.Form{exp.FormSig("qry", []typ.Param{
		{Name: "args"}, {Name: "decls"}, {},
	}), exp.FormResolverFunc(resolvePlan)}
}

func resolvePlan(c *exp.Ctx, env exp.Env, x *exp.Expr, hint typ.Type) (exp.El, error) {
	lo, err := exp.LayoutArgs(x.Rslv.Arg(), x.Args)
	if err != nil {
		return nil, err
	}
	penv := FindEnv(env)
	if penv == nil {
		return nil, cor.Errorf("no plan environment for query %s", x)
	}
	p := penv.Plan
	args := lo.Args(0)
	var rt typ.Type
	if len(args) > 0 {
		// simple query
		if len(lo.Args(1)) > 0 {
			return nil, cor.Errorf("either use simple or compound query got %v rest %v",
				args, lo.Args(1))
		}
		t, err := resolveTask(c, env, exp.Decl{Args: args})
		if err != nil {
			return nil, err
		}
		p.Simple = true
		p.Root = []*Task{t}
		rt = t.Type
	} else {
		decls, err := lo.Decls(1)
		if err != nil {
			return nil, err
		}
		ps := make([]typ.Param, 0, len(decls))
		for _, d := range decls {
			t, err := resolveTask(c, env, d)
			if err != nil {
				return nil, err
			}
			p.Root = append(p.Root, t)
			ps = append(ps, typ.Param{Name: t.Name, Type: t.Type})
		}
		rt = typ.Obj(ps)
	}
	if p.Result == nil {
		// create a synthetic result literal
		p.Result = lit.ZeroProxy(rt)
	} else {
		// compare to expected result
		cmp := typ.Compare(rt, p.Result.Typ())
		if cmp < typ.LvlCheck {
			return nil, cor.Errorf(
				"cannot convert query result type %s to expected result type",
				rt, p.Result.Typ(),
			)
		}
	}
	if len(p.Root) == 0 {
		return nil, cor.Error("empty plan")
	}
	if !c.Exec {
		return x, exp.ErrExec
	}
	err = penv.ExecPlan(c, env, p)
	if err != nil {
		return nil, err
	}
	return p.Result, nil
}

var taskLayout = []typ.Param{{Name: "ref?"}, {Name: "args"}}

func resolveTask(c *exp.Ctx, env exp.Env, d exp.Decl) (*Task, error) {
	lo, err := exp.LayoutArgs(taskLayout, d.Args)
	if err != nil {
		return nil, err
	}
	t := &Task{}
	if d.Name != "" {
		t.Name = d.Name[1:]
	}
	fst := lo.Arg(0)
	if fst == nil {
		// must be field selection in a parent query
		// this transforms +id to (+id .id) an + to (+ .)
		fst = &exp.Sym{Name: "." + t.Name}
	} else if fst.Typ() == typ.Sym {
		switch sym := fst.String(); sym[0] {
		case '?', '*', '#':
			err = resolveQuery(c, env, t, sym, lo)
			if err != nil {
				return nil, err
			}
			return t, nil
		}
	}
	if t.Name == "" {
		return nil, cor.Error("unnamed expr task")
	}
	if len(lo.Args(1)) > 0 || len(lo.Args(2)) > 0 {
		return nil, cor.Error("extra arguments in expr task")
	}
	// partially resolve expression
	fst, err = exp.Resolve(env, fst)
	if err != nil && err != exp.ErrUnres {
		return nil, err
	}
	t.Expr = fst
	if err == nil {
		t.Type = fst.Typ()
		return t, nil
	} else {
		// check for sym, form or func expression to find a result type
		var rt typ.Type
		switch tt := fst.Typ(); tt.Kind {
		case typ.ExpSym:
			rt = fst.(*exp.Sym).Type
		case typ.ExpForm, typ.ExpFunc:
			if x, ok := fst.(*exp.Expr); ok && x.Type != typ.Void {
				rt = x.Type
			} else if tt.Info != nil && len(tt.Params) > 0 {
				rt = tt.Params[len(tt.Params)-1].Type
			}
		}
		switch rt.Kind {
		case typ.KindRef, typ.KindVoid:
		default:
			t.Type = rt
		}
	}
	if t.Type == typ.Void {
		panic(cor.Errorf("no type for task %s %s", d, fst))
	}
	// this is it, we handle the final resolution after planning
	return t, nil
}

var andForm *exp.Form

func init() {
	andForm = exp.Core("and").(*exp.Form)
}
func resolveQuery(c *exp.Ctx, env exp.Env, t *Task, ref string, lo *exp.Layout) error {
	q := &Query{Ref: ref}
	name := ref[1:]
	if name == "" {
		return cor.Error("empty query reference")
	}
	switch name[0] {
	case '.', '/', '$': // path
		return cor.Error("path query reference not yet implemented")
	default:
		// lookup schema
		s := strings.SplitN(name, ".", 3)
		if len(s) < 2 {
			return cor.Errorf("unknown schema name %q", name)
		}
		// locate the project environment and find the model
		pro := dom.FindEnv(env)
		if pro == nil {
			return cor.Error("no project environment")
		}
		m := pro.Schema(s[0]).Model(s[1])
		if m == nil {
			break
		}
		if len(s) > 2 {
			f := m.Field(s[2])
			if f.Param != nil {
				q.Type = f.Type
			}
		} else {
			q.Type = m.Type
		}
	}
	// at this point we need to have the result type to inform argument parsing
	if q.Type == typ.Void {
		return cor.Errorf("no type found for %q", ref)
	}
	args := lo.Args(1)
	tenv := &TaskEnv{env, t, nil}
	sel, err := resolveTag(c, tenv, q, args)
	if err != nil {
		return err
	}
	rt, err := resolveSel(c, tenv, q, sel)
	if err != nil {
		return err
	}
	// TODO check that order only accesses result fields
	t.Query = q
	// set the task result type based on the query subject type
	switch ref[0] {
	case '?':
		t.Type = typ.Opt(rt)
		if q.Lim > 1 {
			return cor.Errorf("unexpected limit %d for single result", q.Lim)
		}
		q.Lim = 1
	case '*':
		t.Type = typ.Arr(rt)
	case '#':
		t.Type = typ.Int
	}
	// simplify where clause
	if len(q.Whr) != 0 {
		x := &exp.Expr{andForm, q.Whr, typ.Bool}
		res, err := exp.Resolve(env, x)
		if err != nil && err != exp.ErrUnres {
			return err
		}
		q.Whr = exp.Dyn{res}
	}

	return nil
}

func resolveTag(c *exp.Ctx, env exp.Env, q *Query, args []exp.El) (sel []exp.El, err error) {
	for _, arg := range args {
		tag, ok := arg.(exp.Tag)
		if !ok {
			q.Whr = append(q.Whr, arg)
			continue
		}
		switch tag.Name {
		case ":whr":
			q.Whr = append(q.Whr, tag.Args...)
		case ":lim":
			// takes one number
			q.Lim, err = resolveInt(c, env, tag.Args)
		case ":off":
			// takes one or two number or a list of two numbers
			q.Off, err = resolveInt(c, env, tag.Args)
		case ":ord", ":asc", ":desc":
			// takes one or more field references
			// can be used multiple times to append to order
			err = resolveOrd(c, env, q, tag.Name == ":desc", tag.Args)
		case "::":
			sel = tag.Args

		default:
			return nil, cor.Errorf("unexpected query tag %q", tag.Name)
		}
		if err != nil {
			return nil, err
		}
	}
	return sel, nil
}

func resolveInt(c *exp.Ctx, env exp.Env, args []exp.El) (int, error) {
	el, err := c.Resolve(env, exp.Dyn(args), typ.Int)
	if err != nil {
		return 0, err
	}
	n, ok := el.(lit.Numer)
	if !ok {
		return 0, cor.Errorf("expect int got %s", el.Typ())
	}
	return int(n.Num()), nil
}

func resolveOrd(c *exp.Ctx, env exp.Env, q *Query, desc bool, args []exp.El) error {
	// either takes a list of strings, one string or one local symbol
	for _, arg := range args {
		sym, ok := arg.(*exp.Sym)
		if !ok {
			return cor.Errorf("want order symbol got %T", arg)
		}
		key := sym.Key()
		if key[0] != '.' {
			key = "." + key
		}
		q.Ord = append(q.Ord, Ord{key, desc})
	}
	return nil
}

func resolveSel(c *exp.Ctx, env exp.Env, q *Query, args []exp.El) (typ.Type, error) {
	var ps []typ.Param
	if q.Type.Kind&typ.MaskElem == typ.KindObj && q.Type.Info != nil {
		ps = q.Type.Params
	}
	// start with all fields unless we start with "-"
	res := make([]*Task, 0, len(ps)+len(args))
	for _, p := range ps {
		res = append(res, &Task{Name: p.Name, Type: p.Type})
	}
	if len(args) == 0 {
		q.Sel = res
		return q.Type, nil
	}
	for i, arg := range args {
		d, ok := arg.(exp.Decl)
		if !ok || d.Name == "" {
			return typ.Void, cor.Errorf("expect declaration got %T", arg)
		}
		switch d.Name[0] {
		case '-':
			if len(d.Name) == 1 {
				if len(d.Args) > 0 {
					// remove all arguments from fields
					keys, err := getKeys(d.Args)
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
				if len(d.Args) > 0 {
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
				if len(d.Args) > 0 {
					if i == 0 { // reset with explicit decls
						res = res[:0]
					}
					// include all arguments as fields
					keys, err := getKeys(d.Args)
					if err != nil {
						return typ.Void, err
					}
					add, err := getParams(ps, keys...)
					if err != nil {
						return typ.Void, err
					}
					res, err = addParams(res, add)
					if err != nil {
						return typ.Void, err
					}
				}
			} else {
				if len(d.Args) > 0 {
					// add arguments as task to sel
					t, err := resolveTask(c, env, d)
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
					res, err = addParams(res, add)
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
	return typ.Obj(ps), nil
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

func addParams(res []*Task, ps []typ.Param) ([]*Task, error) {
	for _, p := range ps {
		for _, t := range res {
			if strings.EqualFold(t.Name, p.Name) {
				return nil, cor.Errorf("key %q already in result %v", p.Key(), res)
			}
		}
		res = append(res, &Task{Name: p.Name, Type: p.Type})
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
