package dom

import (
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
	"github.com/mb0/xelf/utl"
)

var schemaForm, modelForm *exp.Form

func init() {
	schemaForm = &exp.Form{exp.FormSig("schema", []typ.Param{
		{Name: "args"}, {Name: "decls"}, {},
	}), exp.FormResolverFunc(resolveSchema)}
	modelForm = &exp.Form{exp.FormSig("model", []typ.Param{
		{Name: "args"}, {Name: "decls"}, {Name: "tail"}, {},
	}), exp.FormResolverFunc(resolveModel)}
}

func resolveSchema(c *exp.Ctx, env exp.Env, x *exp.Expr, h typ.Type) (exp.El, error) {
	s := &Schema{}
	env = &SchemaEnv{parent: env, Schema: s}
	n, err := utl.NodeResolverFunc(schemaRules, s)(c, env, x, h)
	if err != nil {
		return n, err
	}
	pro := FindEnv(env)
	if pro != nil {
		pro.Schemas = append(pro.Schemas, n.(utl.Node).Ptr().(*Schema))
	}
	return n, nil
}

func resolveModel(c *exp.Ctx, env exp.Env, x *exp.Expr, h typ.Type) (exp.El, error) {
	s := env.(*SchemaEnv)
	m := &Model{Type: typ.Type{typ.KindRec, &typ.Info{}}}
	env = &ModelEnv{SchemaEnv: s, Model: m}
	return utl.NodeResolverFunc(modelRules, m)(c, env, x, h)
}

var schemaRules = utl.NodeRules{
	Tags: utl.TagRules{
		IdxKeyer: utl.OffsetKeyer(2),
		KeyRule: utl.KeyRule{
			KeySetter: utl.ExtraMapSetter("extra"),
		},
	},
	Decl: utl.KeyRule{
		KeyPrepper: func(c *exp.Ctx, env exp.Env, name string, args []exp.El) (lit.Lit, error) {
			tmp := make([]exp.El, 0, len(args)+1)
			tmp = append(tmp, lit.Str(name))
			tmp = append(tmp, args...)
			e, err := resolveModel(c, env, &exp.Expr{modelForm, tmp, typ.Void}, typ.Void)
			if err != nil {
				return nil, err
			}
			return e.(lit.Lit), nil
		},
		KeySetter: func(n utl.Node, key string, el lit.Lit) error {
			m := el.(utl.Node).Ptr().(*Model)
			s := n.Ptr().(*Schema)
			m.schema = s.Key()
			m.Type.Ref = m.schema + "." + m.Name
			s.Models = append(s.Models, m)
			return nil
		},
	},
}

var modelRules = utl.NodeRules{
	Tags: utl.TagRules{
		IdxKeyer: utl.OffsetKeyer(2),
		KeyRule: utl.KeyRule{
			KeySetter: utl.ExtraMapSetter("extra"),
		},
		Rules: map[string]utl.KeyRule{
			"typ": {KeyPrepper: typPrepper, KeySetter: typSetter},
			"idx": {KeyPrepper: idxPrepper, KeySetter: idxSetter},
		},
	},
	Decl: utl.KeyRule{
		KeyPrepper: func(c *exp.Ctx, env exp.Env, key string, args []exp.El) (lit.Lit, error) {
			m := env.(*ModelEnv)
			switch m.Model.Kind {
			case typ.KindFlag, typ.KindEnum:
				return resolveConst(c, m, key, args)
			case typ.KindRec, typ.ExpFunc:
				return resolveField(c, m, key, args)
			}
			return nil, cor.Errorf("unexpected model kind %s", m.Model.Kind)
		},
		KeySetter: noopSetter,
	},
}

func noopSetter(n utl.Node, key string, el lit.Lit) error { return nil }

func resolveConst(c *exp.Ctx, env *ModelEnv, key string, args []exp.El) (lit.Lit, error) {
	d, err := resolveConstVal(c, env, args, len(env.Model.Consts))
	if err != nil {
		return nil, cor.Errorf("resolve const val: %w", err)
	}
	m := env.Model
	m.Consts = append(m.Consts, cor.Const{key, int64(d)})
	m.Elems = append(m.Elems, &Elem{})
	return m.Type, nil
}

func resolveConstVal(c *exp.Ctx, env *ModelEnv, args []exp.El, idx int) (_ lit.Int, err error) {
	var el exp.El
	switch len(args) {
	case 0:
		if env.Model.Kind&typ.MaskRef == typ.KindFlag {
			return lit.Int(1 << uint64(idx)), nil
		}
		return lit.Int(idx) + 1, nil
	case 1:
		el, err = c.Resolve(env, args[0], typ.Int)
	default:
		el, err = c.Resolve(env, exp.Dyn(args), typ.Int)
	}
	if err != nil {
		return 0, err
	}
	n, ok := el.(lit.Num)
	if !ok {
		return 0, cor.Errorf("expect num got %T", el)
	}
	return lit.Int(n.Num()), nil
}

var bitRule = utl.KeyRule{
	KeyPrepper: utl.FlagPrepper(cor.Consts(bitConsts)),
	KeySetter:  utl.FlagSetter("bits"),
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
		"typ":  {KeyPrepper: typPrepper, KeySetter: typSetter},
	},
	KeyRule: utl.KeyRule{KeySetter: utl.ExtraMapSetter("extra")},
}

func resolveField(c *exp.Ctx, env *ModelEnv, name string, args []exp.El) (lit.Lit, error) {
	p, el := typ.Param{Name: name}, &Elem{}
	if strings.HasSuffix(name, "?") {
		el.Bits = BitOpt
	}
	err := utl.ParseTags(c, env, args, &FieldElem{&p, el}, fieldRules)
	if err != nil {
		return nil, cor.Errorf("parsing tags: %w", err)
	}
	m := env.Model
	m.Elems = append(m.Elems, el)
	m.Params = append(m.Params, p)
	return p.Type, nil
}

func typPrepper(c *exp.Ctx, env exp.Env, key string, args []exp.El) (_ lit.Lit, err error) {
	if len(args) == 0 {
		return nil, cor.Errorf("expect type for model kind")
	}
	fst := args[0]
	fst, err = c.Resolve(env, fst, typ.Typ)
	if err != nil && err != exp.ErrUnres {
		return nil, err
	}
	if t, ok := fst.(typ.Type); ok {
		return t, nil
	}
	return nil, cor.Errorf("expect type for model kind, got %T", args[0])
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

func idxPrepper(c *exp.Ctx, env exp.Env, key string, args []exp.El) (lit.Lit, error) {
	l, err := utl.DynPrepper(c, env, key, args)
	if err != nil {
		return l, cor.Errorf("dyn prepper: %w", err)
	}
	uniq := key == "uniq"
	k := l.Typ().Kind
	if k&typ.BaseList != 0 {
		return &lit.Dict{List: []lit.Keyed{{"keys", l}, {"unique", lit.Bool(uniq)}}}, nil
	}
	return l, nil
}
func idxSetter(o utl.Node, key string, l lit.Lit) error {
	m := o.Ptr().(*Model)
	var idx Index
	err := lit.AssignTo(l, &idx)
	if err != nil {
		return cor.Errorf("assign idx to %s: %w", l, err)
	}
	if m.Rec == nil {
		m.Rec = &Record{}
	}
	m.Rec.Indices = append(m.Rec.Indices, &idx)
	return nil
}
