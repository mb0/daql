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

var schemaRules = utl.NodeRules{}

func resolveSchema(c *exp.Ctx, env *SchemaEnv, name string, els []exp.El) error {
	s := &Schema{Name: name}
	n, err := utl.GetNode(s)
	if err != nil {
		return err
	}
	env.Schema = s
	env.Node = n
	env.mm = make(map[string]*ModelEnv)

	r := schemaRules
	if s.Name != "" {
		r.Tags = r.Tags.WithOffset(1)
	}
	r.Decl.KeyPrepper = func(c *exp.Ctx, _ exp.Env, name string, args []exp.El) (lit.Lit, error) {
		m := &ModelEnv{SchemaEnv: env}
		p, err := resolveModel(c, m, name, args)
		if err == nil {
			env.mm[m.Model.Key()] = m
		}
		return p, err
	}
	r.Decl.KeySetter = func(n utl.Node, key string, el lit.Lit) error {
		m := el.(utl.Node).Ptr().(*Model)
		m.schema = s.Name
		s.Models = append(s.Models, m)
		return nil

	}
	return utl.ParseDeclNode(c, env, els, n, r)
}

var modelRules = utl.NodeRules{
	Tags: utl.TagRules{
		IdxKeyer: utl.OffsetKeyer(1),
		KeyRule: utl.KeyRule{
			KeySetter: utl.ExtraMapSetter("extra"),
		},
		Rules: map[string]utl.KeyRule{
			"kind": {KeyPrepper: kindPrepper},
			"idx":  {KeyPrepper: idxPrepper, KeySetter: idxSetter},
		},
	},
}

func resolveModel(c *exp.Ctx, env *ModelEnv, name string, els []exp.El) (lit.Lit, error) {
	m := &Model{Name: name, Kind: typ.KindRec}
	p, err := utl.GetNode(m)
	if err != nil {
		return nil, err
	}
	env.Model = m
	r := modelRules
	r.Decl.KeyPrepper = func(c *exp.Ctx, _ exp.Env, key string, args []exp.El) (lit.Lit, error) {
		if m.Kind != typ.KindRec {
			return resolveConst(c, env, key, args)
		}
		return resolveField(c, env, key, args)
	}
	r.Decl.KeySetter = func(n utl.Node, key string, el lit.Lit) error {
		p := el.(utl.Node).Ptr()
		if m.Kind != typ.KindRec {
			m.Consts = append(m.Consts, *(p.(*cor.Const)))
		} else {
			m.Fields = append(m.Fields, p.(*Field))
		}
		return nil

	}
	err = utl.ParseDeclNode(c, env, els, p, r)
	if err != nil {
		return nil, err
	}
	return p, nil
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
