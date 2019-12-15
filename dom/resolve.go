package dom

import (
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/prx"
	"github.com/mb0/xelf/std"
	"github.com/mb0/xelf/typ"
	"github.com/mb0/xelf/utl"
)

func splitDecls(args []exp.El) (tags, decl []*exp.Tag) {
	tags = make([]*exp.Tag, 0, len(args))
	decl = make([]*exp.Tag, 0, len(args))
	for _, arg := range args {
		switch t := arg.(type) {
		case *exp.Tag:
			if t.Name != "" && cor.IsKey(t.Name) && t.Name[0] != '_' {
				tags = append(tags, t)
			} else {
				decl = append(decl, t)
			}
		default:
			decl = append(decl, &exp.Tag{El: arg, Src: arg.Source()})
		}
	}
	return tags, decl
}

var projectSpec = std.SpecXX("<form project name:sym tail?; @>",
	func(x std.CallCtx) (exp.El, error) {
		p := FindEnv(x.Env)
		n, err := utl.GetNode(p.Project)
		if err != nil {
			return nil, err
		}
		sym, ok := x.Arg(0).(*exp.Sym)
		if !ok {
			return nil, cor.Errorf("expect project symbol got %T", x.Arg(0))
		}
		p.Name = sym.Name
		tags, decl := splitDecls(x.Args(1))
		err = commonRules.Resolve(x.Prog, x.Env, tags, n)
		if err != nil {
			return nil, err
		}
		if p.Schemas == nil {
			p.Schemas = make([]*Schema, 0, len(decl))
		}
		for _, d := range decl {
			name := d.Key()
			s := &Schema{Common: Common{Name: name}}
			_, err = resolveSchema(x.Prog, x.Env, s, d.Args())
			if err != nil {
				return nil, err
			}
			p.Schemas = append(p.Schemas, s)
		}
		return &exp.Atom{Lit: n}, nil
	})

var schemaSpec = std.SpecXX("<form schema name:sym tail?; @>",
	func(x std.CallCtx) (exp.El, error) {
		s := &Schema{Common: Common{Extra: &lit.Dict{}}}
		sym, ok := x.Arg(0).(*exp.Sym)
		if !ok {
			return nil, cor.Errorf("expect schema symbol got %T %s", x.Arg(0), x.Arg(0))
		}
		s.Name = sym.Name
		n, err := resolveSchema(x.Prog, x.Env, s, x.Args(1))
		if err != nil {
			return nil, err
		}
		pro := FindEnv(x.Env)
		if pro != nil {
			pro.Schemas = append(pro.Schemas, s)
		}
		return &exp.Atom{Lit: n}, nil
	})

func resolveSchema(p *exp.Prog, env exp.Env, s *Schema, args []exp.El) (utl.Node, error) {
	n, err := utl.GetNode(s)
	if err != nil {
		return nil, err
	}
	senv := &SchemaEnv{parent: env, Schema: s}
	tags, decl := splitDecls(args)
	err = commonRules.Resolve(p, senv, tags, n)
	if err != nil {
		return nil, cor.Errorf("schema common rules: %v", err)
	}
	qual := s.Key()
	// first initialize the models...
	s.Models = make([]*Model, 0, len(decl))
	for _, d := range decl {
		m := &Model{
			Common: Common{Name: d.Name}, Schema: qual,
			Type: typ.Type{typ.KindObj, &typ.Info{
				Ref: qual + "." + d.Name,
			}},
		}
		s.Models = append(s.Models, m)
	}
	// ...then resolve the models with all other schema model names in scope
	for i, m := range s.Models {
		err = resolveModel(p, senv, m, decl[i].Args())
		if err != nil {
			return nil, err
		}
	}
	return n, nil
}

func resolveModel(p *exp.Prog, env *SchemaEnv, m *Model, args []exp.El) error {
	if len(args) == 0 {
		return cor.Errorf("empty model arguments")
	}
	menv := &ModelEnv{SchemaEnv: env, Model: m}
	n, err := utl.GetNode(m)
	if err != nil {
		return err
	}
	at, ok := args[0].(*exp.Atom)
	if !ok || at.Typ().Kind != typ.KindTyp {
		return cor.Errorf("expect model kind got %T", args[0])
	}
	m.Type.Kind = at.Lit.(typ.Type).Kind
	tags, decl := splitDecls(args[1:])
	err = modelRules.Resolve(p, menv, tags, n)
	if err != nil {
		return err
	}
	for _, d := range decl {
		switch m.Type.Kind {
		case typ.KindBits, typ.KindEnum:
			_, err = resolveConst(p, menv, d)
		case typ.KindObj, typ.KindFunc:
			_, err = resolveField(p, menv, d)
		default:
			err = cor.Errorf("unexpected model kind %s", m.Type.Kind)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

var commonRules = utl.TagRules{
	IdxKeyer: utl.OffsetKeyer(0),
	KeyRule:  utl.KeyRule{KeySetter: utl.ExtraMapSetter("extra")},
}

var modelRules = utl.TagRules{
	IdxKeyer: utl.OffsetKeyer(2),
	KeyRule:  utl.KeyRule{KeySetter: utl.ExtraMapSetter("extra")},
	Rules: map[string]utl.KeyRule{
		"type": {typPrepper, typSetter},
		"idx":  {idxPrepper, idxSetter},
	},
}
var defaultRules utl.TagRules

func resolveConst(p *exp.Prog, env *ModelEnv, n *exp.Tag) (lit.Lit, error) {
	args := n.Args()
	d, err := resolveConstVal(p, env, args, len(env.Model.Type.Consts))
	if err != nil {
		return nil, cor.Errorf("resolve const val: %w", err)
	}
	m := env.Model
	m.Type.Consts = append(m.Type.Consts, typ.Const{n.Name, int64(d)})
	m.Elems = append(m.Elems, &Elem{})
	return m.Type, nil
}

func resolveConstVal(p *exp.Prog, env *ModelEnv, args []exp.El, idx int) (_ lit.Int, err error) {
	var el exp.El
	switch len(args) {
	case 0:
		if env.Model.Type.Kind&typ.MaskRef == typ.KindBits {
			return lit.Int(1 << uint64(idx)), nil
		}
		return lit.Int(idx) + 1, nil
	case 1:
		el, err = p.Eval(env, args[0], typ.Int)
	default:
		el, err = p.Eval(env, &exp.Dyn{Els: args}, typ.Int)
	}
	if err != nil {
		return 0, err
	}
	n, ok := el.(*exp.Atom).Lit.(lit.Num)
	if !ok {
		return 0, cor.Errorf("expect num got %T", el)
	}
	return lit.Int(n.Num()), nil
}

var bitRule = utl.KeyRule{
	KeyPrepper: utl.BitsPrepper(typ.Constants(bitConsts)),
	KeySetter:  utl.BitsSetter("bits"),
}
var fieldRules = utl.TagRules{
	IdxKeyer: utl.OffsetKeyer(1),
	Rules: map[string]utl.KeyRule{
		"pk":   bitRule,
		"idx":  bitRule,
		"uniq": bitRule,
		"ordr": bitRule,
		"auto": bitRule,
		"ro":   bitRule,
		"type": {KeyPrepper: typPrepper, KeySetter: typSetter},
	},
	KeyRule: utl.KeyRule{KeySetter: utl.ExtraMapSetter("extra")},
}

func resolveField(p *exp.Prog, env *ModelEnv, n *exp.Tag) (lit.Lit, error) {
	name := n.Name
	if name == "_" {
		name = ""
	}
	param, el := typ.Param{Name: name}, &Elem{}
	if strings.HasSuffix(n.Name, "?") {
		el.Bits = BitOpt
	}
	err := utl.ParseTags(p, env, n.Args(), &FieldElem{&param, el}, fieldRules)
	if err != nil {
		return nil, cor.Errorf("parsing tags for %q: %w", n.Name, err)
	}
	m := env.Model
	m.Elems = append(m.Elems, el)
	m.Type.Params = append(m.Type.Params, param)
	return param.Type, nil
}

func typPrepper(p *exp.Prog, env exp.Env, n *exp.Tag) (_ lit.Lit, err error) {
	args := n.Args()
	if len(args) == 0 {
		return nil, cor.Errorf("expect type for model kind")
	}
	fst := args[0]
	fst, err = p.Eval(env, fst, typ.Void)
	if err != nil && err != exp.ErrUnres {
		return nil, cor.Errorf("prep type %v", err)
	}
	if a, ok := fst.(*exp.Atom); ok {
		if t, ok := a.Lit.(typ.Type); ok {
			return t, nil
		}
	}
	if s, ok := fst.(*exp.Sym); ok && s.Type != typ.Void {
		return s.Type, nil
	}
	return nil, cor.Errorf("expect type, got %q %T", fst, fst)
}
func typSetter(o utl.Node, key string, l lit.Lit) error {
	switch m := o.Ptr().(type) {
	case *Model:
		m.Type.Kind = l.(typ.Type).Kind
	case *FieldElem:
		m.Type = l.(typ.Type)
	default:
		return cor.Errorf("unexpected node %T for %s", o, key)
	}
	return nil
}

func idxPrepper(p *exp.Prog, env exp.Env, n *exp.Tag) (lit.Lit, error) {
	l, err := utl.DynPrepper(p, env, n)
	if err != nil {
		return l, cor.Errorf("dyn prepper: %w", err)
	}
	uniq := n.Key() == "uniq"
	k := l.Typ().Kind
	if k&typ.KindIdxr != 0 {
		return &lit.Dict{List: []lit.Keyed{{"keys", l}, {"unique", lit.Bool(uniq)}}}, nil
	}
	return l, nil
}
func idxSetter(o utl.Node, key string, l lit.Lit) error {
	m := o.Ptr().(*Model)
	var idx Index
	err := prx.AssignTo(l, &idx)
	if err != nil {
		return cor.Errorf("assign idx to %s: %w", l, err)
	}
	if m.Object == nil {
		m.Object = &Object{}
	}
	m.Object.Indices = append(m.Object.Indices, &idx)
	return nil
}
