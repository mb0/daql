package qry

import (
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/std"
	"github.com/mb0/xelf/typ"
)

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

func simpleExpr(el exp.El) (string, exp.El) {
	s, ok := el.(*exp.Sym)
	if ok && cor.IsKey(s.Name) {
		n := *s
		n.Name = "." + s.Name
		return s.Name, &n
	}
	return "", el
}

func isQueryRef(a exp.El) bool {
	s, ok := a.(*exp.Sym)
	if !ok || s.Name == "" {
		return false
	}
	switch s.Name[0] {
	case '?', '*', '#':
		return true
	}
	return false
}

var qrySpec = std.SpecXX("<form qry tail?; @>", func(x std.CallCtx) (exp.El, error) {
	qenv := FindEnv(x.Env)
	if qenv == nil {
		return nil, cor.Errorf("no qry environment for query %s", x)
	}
	args := x.Args(0)
	if len(args) == 0 {
		return nil, cor.Errorf("empty query")
	}
	doc := &Doc{}
	penv := exp.NewParamReslEnv(x.Env, x.Ctx)
	denv := doc.ReslEnv(penv)
	if isQueryRef(args[0]) {
		t, err := resolveTask(x.Prog, denv, "", args, nil)
		if err != nil {
			return nil, err
		}
		doc.Root = []*Task{t}
		doc.Type = t.Type
	} else {
		decl := x.Tags(0)
		ps := make([]typ.Param, 0, len(decl))
		for _, d := range decl {
			args := d.Args()
			t, err := resolveTask(x.Prog, denv, d.Name, args, nil)
			if err != nil {
				return nil, err
			}
			doc.Root = append(doc.Root, t)
			ps = append(ps, typ.Param{Name: t.Name, Type: t.Type})
		}
		doc.Type = typ.Rec(ps)
	}
	return &exp.Atom{Lit: &exp.Spec{typ.Func("", []typ.Param{
		{"arg", typ.Any},
		{"", doc.Type},
	}), doc}}, nil
})

var taskSig = exp.MustSig("<form task tail?; @>")

func declName(args []exp.El) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	fst, ok := args[0].(*exp.Sym)
	if !ok {
		return "", false
	}
	return fst.Name, true
}

func resolveTask(p *exp.Prog, env exp.Env, name string, args []exp.El, par *Task) (t *Task, err error) {
	t = &Task{Parent: par, Name: name}
	var fst exp.El
	if len(args) == 0 {
		fst = &exp.Sym{Name: "." + t.Name}
	} else {
		fst = args[0]
		switch sym := fst.String(); sym[0] {
		case '?', '*', '#':
			err = resolveQuery(p, env, t, sym, args[1:])
			if err != nil {
				return nil, err
			}
			return t, nil
		default:
			fst = &exp.Dyn{Els: args}
		}
	}
	// partially resolve expression
	fst, err = exp.Resl(env, fst)
	if fst == nil {
		return nil, cor.Errorf("resolve task %s: %v", args, err)
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
		return nil, cor.Errorf("no type for task %s, %s in %T", args, fst, env)
	}
	// this is it, we handle the final resolution after planning
	return t, nil
}

var andSpec = std.Core("and")

func resolveQuery(p *exp.Prog, env exp.Env, t *Task, ref string, args []exp.El) error {
	q := &Query{Ref: ref}
	name := ref[1:]
	if name == "" {
		return cor.Error("empty query reference")
	}
	// locate the plan environment for a project and find the model
	penv := FindEnv(env)
	switch name[0] {
	case '.', '/', '$': // path
		return cor.Error("path query reference not yet implemented")
	default:
		// lookup schema
		s := strings.SplitN(name, ".", 2)
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
	}
	// at this point we need to have the result type to inform argument parsing
	if q.Type == typ.Void {
		return cor.Errorf("no type found for %q", ref)
	}
	t.Query = q
	tenv := &SelEnv{env, t}
	whr, args := splitPlain(args)
	tags, decl := splitDecls(args)
	err := resolveTag(p, tenv, q, tags)
	if err != nil {
		return err
	}
	if len(whr) > 0 {
		we := t.Query.Whr
		if we != nil {
			we.Els = append(we.Els, whr...)
		} else {
			t.Query.Whr = &exp.Dyn{Els: whr}
		}
	}
	// TODO check that order only accesses result fields
	rt, err := resolveSel(p, tenv, q, decl)
	if err != nil {
		return err
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

func resolveTag(p *exp.Prog, env exp.Env, q *Query, tags []*exp.Tag) (err error) {
	var whr []exp.El
	if q.Whr != nil {
		whr = q.Whr.Els
	}
	for _, tag := range tags {
		switch tag.Name {
		case "whr":
			whr = append(whr, tag.El)
		case "lim":
			// takes one number
			q.Lim, err = resolveInt(p, env, tag.El)
		case "off":
			// takes one or two number or a list of two numbers
			q.Off, err = resolveInt(p, env, tag.El)
		case "ord", "asc", "desc":
			// takes one or more field references
			// can be used multiple times to append to order
			_, el := simpleExpr(tag.El)
			err = resolveOrd(p, env, q, tag.Name == "desc", el)
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

func resolveInt(p *exp.Prog, env exp.Env, arg exp.El) (int64, error) {
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

func resolveOrd(p *exp.Prog, env exp.Env, q *Query, desc bool, arg exp.El) error {
	// either takes a list of strings, one string or one local symbol
	sym, ok := arg.(*exp.Sym)
	if !ok {
		return cor.Errorf("want order symbol got %T", arg)
	}
	q.Ord = append(q.Ord, Ord{"." + sym.Key(), desc})
	return nil
}

func resolveSel(p *exp.Prog, env *SelEnv, q *Query, args []*exp.Tag) (typ.Type, error) {
	var ps []typ.Param
	if q.Type.Kind&typ.MaskElem == typ.KindRec && q.Type.HasParams() {
		ps = q.Type.Params
	}
	res := make([]*Task, 0, len(ps)+len(args))
	for _, p := range ps {
		res = append(res, &Task{Name: p.Name, Type: p.Type, Parent: env.Task})
	}
	if len(args) == 0 {
		q.Sel = res
		return q.Type, nil
	}
	var mode byte
	var err error
	for i, d := range args {
		args := d.Args()
		name := d.Name
		switch name {
		case "+", "-":
			mode = name[0]
			if d.El != nil {
				return typ.Void, cor.Errorf("unexpected selection arguments %s", d)
			}
			continue
		case "_":
			mode = '+'
			if d.El == nil {
				res = res[:0]
				continue
			} else if len(args[i:]) > 1 {
				return typ.Void, cor.Errorf("unexpected selection arguments %s", d)
			}
			var sub string
			sub, d.El = simpleExpr(d.El)
			t, err := resolveTask(p, env, sub, d.Args(), env.Task)
			if err != nil {
				return typ.Void, err
			}
			res = append(res[:0], t)
			q.Sel = res
			q.Sca = true
			return t.Type, nil
		case "":
			return typ.Void, cor.Errorf("unnamed selection %s", d)
		}
		switch name[0] {
		case '-', '+':
			mode = name[0]
			name = name[1:]
		default:
		}
		key := strings.ToLower(name)
		switch mode {
		case '-': // exclude
			if d.El != nil {
				return typ.Void, cor.Errorf("unexpected selection arguments %s", d)
			}
			res, err = removeKey(res, key)
			if err != nil {
				return typ.Void, err
			}
		case '+':
			if d.El == nil { // naked selects choose a subj field by key
				add, err := getParams(ps, key)
				if err != nil {
					return typ.Void, err
				}
				res, err = addParams(res, add, env.Task)
				if err != nil {
					return typ.Void, err
				}
			} else {
				t, err := resolveTask(p, env, name, d.Args(), env.Task)
				if err != nil {
					return typ.Void, err
				}
				res = append(res, t)
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
