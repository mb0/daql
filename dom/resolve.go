package dom

import (
	"fmt"
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
	m := &Model{Kind: typ.KindRec}
	env = &ModelEnv{SchemaEnv: s, Model: m}
	return utl.NodeResolverFunc(modelRules, m)(c, env, x, h)
}

var schemaRules = utl.NodeRules{
	Tags: utl.TagRules{IdxKeyer: utl.ZeroKeyer},
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
			m.schema = s.Name
			s.Models = append(s.Models, m)
			return nil
		},
	},
}

var modelRules = utl.NodeRules{
	Tags: utl.TagRules{
		IdxKeyer: utl.ZeroKeyer,
		KeyRule: utl.KeyRule{
			KeySetter: utl.ExtraMapSetter("extra"),
		},
		Rules: map[string]utl.KeyRule{
			"kind": {KeyPrepper: kindPrepper},
			"idx":  {KeyPrepper: idxPrepper, KeySetter: idxSetter},
		},
	},
	Decl: utl.KeyRule{
		KeyPrepper: func(c *exp.Ctx, env exp.Env, key string, args []exp.El) (lit.Lit, error) {
			m := env.(*ModelEnv)
			if m.Model.Kind != typ.KindRec {
				return resolveConst(c, m, key, args)
			}
			return resolveField(c, m, key, args)

		},
		KeySetter: func(n utl.Node, key string, el lit.Lit) error {
			p := el.(utl.Node).Ptr()
			m := n.Ptr().(*Model)
			if m.Kind != typ.KindRec {
				m.Consts = append(m.Consts, *(p.(*cor.Const)))
			} else {
				m.Fields = append(m.Fields, p.(*Field))
			}
			return nil
		},
	},
}

func resolveConst(c *exp.Ctx, env *ModelEnv, key string, args []exp.El) (lit.Lit, error) {
	d, err := resolveConstVal(c, env, args, len(env.Model.Consts))
	if err != nil {
		return nil, err
	}
	return utl.GetNode(&cor.Const{key, int64(d)})
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
		return 0, fmt.Errorf("expect num got %T", el)
	}
	return lit.Int(n.Num()), nil
}

var bitRule = utl.KeyRule{
	KeyPrepper: utl.FlagPrepper(bitConsts),
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
	},
	KeyRule: utl.KeyRule{KeySetter: utl.ExtraMapSetter("extra")},
}

func resolveField(c *exp.Ctx, env *ModelEnv, name string, args []exp.El) (lit.Lit, error) {
	f := &Field{Name: name}
	if strings.HasSuffix(name, "?") {
		f.Name = name[:len(name)-1]
		f.Bits = BitOpt
	}
	err := utl.ParseTags(c, env, args, f, fieldRules)
	if err != nil {
		return nil, err
	}
	return utl.GetNode(f)
}

func kindPrepper(c *exp.Ctx, env exp.Env, key string, args []exp.El) (lit.Lit, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("expect type for model kind")
	}
	if s, ok := args[0].(*exp.Sym); ok {
		t, err := typ.ParseSym(s.Name, nil)
		if err == nil {
			return lit.Int(t.Kind), nil
		}
	}
	return nil, fmt.Errorf("expect type for model kind, got %T", args[0])
}

func idxPrepper(c *exp.Ctx, env exp.Env, key string, args []exp.El) (lit.Lit, error) {
	l, err := utl.DynPrepper(c, env, key, args)
	if err != nil {
		return l, err
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
		return fmt.Errorf("%v to *Index: %s", err, l)
	}
	m.Indices = append(m.Indices, &idx)
	return nil
}
