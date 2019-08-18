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

var projectSpec = std.SpecXX("(form 'project' :args? :decls? : @)",
	func(x std.CallCtx) (exp.El, error) {
		p := FindEnv(x.Env)
		n, err := utl.GetNode(p.Project)
		if err != nil {
			return nil, err
		}
		err = commonRules.Resolve(x.Prog, x.Env, x.Tags(0), n)
		if err != nil {
			return nil, err
		}
		decls, err := x.Decls(1)
		if err != nil {
			return nil, err
		}
		if p.Schemas == nil {
			p.Schemas = make([]*Schema, 0, len(decls))
		}
		for _, d := range decls {
			name := d.Name[1:]
			s := &Schema{Common: Common{Name: name}}
			slo, err := exp.FormLayout(schemaSpec.Type, d.Args())
			if err != nil {
				return nil, err
			}
			_, err = resolveSchema(x.Prog, x.Env, s, slo)
			if err != nil {
				return nil, err
			}
			p.Schemas = append(p.Schemas, s)
		}
		return &exp.Atom{Lit: n}, nil
	})

var schemaSpec = std.SpecXX("(form 'schema' :args? :decls? : @)",
	func(x std.CallCtx) (exp.El, error) {
		s := &Schema{Common: Common{Extra: &lit.Dict{}}}
		n, err := resolveSchema(x.Prog, x.Env, s, &x.Layout)
		if err != nil {
			return nil, err
		}
		pro := FindEnv(x.Env)
		if pro != nil {
			pro.Schemas = append(pro.Schemas, s)
		}
		return &exp.Atom{Lit: n}, nil
	})

func resolveSchema(p *exp.Prog, env exp.Env, s *Schema, lo *exp.Layout) (utl.Node, error) {
	n, err := utl.GetNode(s)
	if err != nil {
		return nil, err
	}
	senv := &SchemaEnv{parent: env, Schema: s}
	err = commonRules.Resolve(p, senv, lo.Tags(0), n)
	if err != nil {
		return nil, err
	}
	decls, err := lo.Decls(1)
	if err != nil {
		return nil, err
	}
	qual := s.Key()
	// first initialize the models...
	s.Models = make([]*Model, 0, len(decls))
	for _, d := range decls {
		name := d.Name[1:]
		m := &Model{
			Common: Common{Name: name}, Schema: qual,
			Type: typ.Type{typ.KindObj, &typ.Info{
				Ref: qual + "." + name,
			}},
		}
		s.Models = append(s.Models, m)
	}
	// ...then resolve the models with all other schema model names in scope
	for i, m := range s.Models {
		err = resolveModel(p, senv, m, decls[i].Args())
		if err != nil {
			return nil, err
		}
	}
	return n, nil
}

func resolveModel(p *exp.Prog, env *SchemaEnv, m *Model, args []exp.El) error {
	menv := &ModelEnv{SchemaEnv: env, Model: m}
	n, err := utl.GetNode(m)
	if err != nil {
		return err
	}
	lo, err := exp.FormLayout(modelSig, args)
	if err != nil {
		return err
	}
	err = modelRules.Resolve(p, menv, lo.Tags(0), n)
	if err != nil {
		return err
	}
	decls, err := lo.Decls(1)
	if err != nil {
		return err
	}
	for _, d := range decls {
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
	return defaultRules.Resolve(p, menv, lo.Tags(2), n)
}

var modelSig = exp.MustSig("(form 'model' :args? :decls? :tail? : @)")

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

func resolveConst(p *exp.Prog, env *ModelEnv, n *exp.Named) (lit.Lit, error) {
	d, err := resolveConstVal(p, env, n.Args(), len(env.Model.Type.Consts))
	if err != nil {
		return nil, cor.Errorf("resolve const val: %w", err)
	}
	m := env.Model
	m.Type.Consts = append(m.Type.Consts, typ.Const{n.Name[1:], int64(d)})
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

func resolveField(p *exp.Prog, env *ModelEnv, n *exp.Named) (lit.Lit, error) {
	param, el := typ.Param{Name: n.Name[1:]}, &Elem{}
	if strings.HasSuffix(n.Name, "?") {
		el.Bits = BitOpt
	}
	err := utl.ParseTags(p, env, n.Args(), &FieldElem{&param, el}, fieldRules)
	if err != nil {
		return nil, cor.Errorf("parsing tags: %w", err)
	}
	m := env.Model
	m.Elems = append(m.Elems, el)
	m.Type.Params = append(m.Type.Params, param)
	return param.Type, nil
}

func typPrepper(p *exp.Prog, env exp.Env, n *exp.Named) (_ lit.Lit, err error) {
	args := n.Args()
	if len(args) == 0 {
		return nil, cor.Errorf("expect type for model kind")
	}
	fst := args[0]
	fst, err = p.Eval(env, fst, typ.Void)
	if err != nil && err != exp.ErrUnres {
		return nil, err
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

func idxPrepper(p *exp.Prog, env exp.Env, n *exp.Named) (lit.Lit, error) {
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
