package dom

import (
	"fmt"
	"strings"

	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

func (Bit) Bits() map[string]int64 { return bitConsts }

func (c *Common) Key() string { return strings.ToLower(c.Name) }

type Node interface {
	Qualified() string
	String() string
	WriteBfr(b *bfr.Ctx) error
}

func (m *Model) Qual() string      { return m.Schema }
func (m *Model) Qualified() string { return fmt.Sprintf("%s.%s", m.Schema, m.Key()) }

func (s *Schema) Qualified() string { return s.Key() }

func (p *Project) Qualified() string { return fmt.Sprintf("_%s", p.Key()) }

// Schema returns a schema for key or nil.
func (p *Project) Schema(key string) *Schema {
	if p != nil {
		for _, s := range p.Schemas {
			if s.Name == key {
				return s
			}
		}
	}
	return nil
}

// Model returns a model for the qualified key or nil.
func (p *Project) Model(key string) *Model {
	split := strings.SplitN(key, ".", 2)
	if len(split) == 2 {
		return p.Schema(split[0]).Model(split[1])
	}
	return nil
}

// Model returns a model for key or nil.
func (s *Schema) Model(key string) *Model {
	if s != nil {
		for _, m := range s.Models {
			if m.Key() == key {
				return m
			}
		}
	}
	return nil
}

type ConstElem struct {
	*typ.Const
	*Elem
}

// Const returns a constant element for key or nil.
func (m *Model) Const(key string) ConstElem {
	if m != nil {
		for i, c := range m.Type.Consts {
			if c.Key() == key {
				return ConstElem{&m.Type.Consts[i], m.Elems[i]}
			}
		}
	}
	return ConstElem{}
}

type FieldElem struct {
	*typ.Param
	*Elem
}

// Field returns a field element for key or nil.
func (m *Model) Field(key string) FieldElem {
	if m != nil {
		_, i, err := m.Type.ParamByKey(key)
		if err == nil {
			return FieldElem{&m.Type.Params[i], m.Elems[i]}
		}
	}
	return FieldElem{}
}

var bitConsts = map[string]int64{
	"Opt":  int64(BitOpt),
	"PK":   int64(BitPK),
	"Idx":  int64(BitIdx),
	"Uniq": int64(BitUniq),
	"Ordr": int64(BitOrdr),
	"Auto": int64(BitAuto),
	"RO":   int64(BitRO),
}

func setNode(n *Common, x lit.Keyed) error {
	switch x.Key {
	case "name":
		n.Name = x.Lit.(lit.Character).Char()
	default:
		if n.Extra == nil {
			n.Extra = &lit.Dict{}
		}
		_, err := n.Extra.SetKey(x.Key, x.Lit)
		return err
	}
	return nil
}

func addElemFromDict(m *Model, d *lit.Dict) error {
	var el Elem
	var p typ.Param
	var c typ.Const
	for _, x := range d.List {
		switch x.Key {
		case "name":
			p.Name = x.Lit.(lit.Character).Char()
			c.Name = p.Name
		case "val":
			c.Val = int64(x.Lit.(lit.Numeric).Num())
		case "ref":
			el.Ref = x.Lit.(lit.Character).Char()
		case "type":
			t, err := typ.ParseSym(x.Lit.(lit.Character).Char(), nil)
			if err != nil {
				return err
			}
			p.Type = t
		case "bits":
			el.Bits = Bit(x.Lit.(lit.Numeric).Num())
		default:
			if el.Extra == nil {
				el.Extra = &lit.Dict{}
			}
			_, err := el.Extra.SetKey(x.Key, x.Lit)
			return err
		}
	}
	if m.Type.Kind&typ.KindPrim != 0 {
		m.Type.Consts = append(m.Type.Consts, c)
	} else {
		m.Type.Params = append(m.Type.Params, p)
	}
	m.Elems = append(m.Elems, &el)
	return nil
}

func (m *Model) FromDict(d *lit.Dict) (err error) {
	if m.Type.Info == nil {
		m.Type.Kind = typ.KindObj
		m.Type.Info = &typ.Info{}
	}
	for _, x := range d.List {
		switch x.Key {
		case "type":
			t, err := typ.ParseSym(x.Lit.(lit.Character).Char(), nil)
			if err != nil {
				return err
			}
			m.Type.Kind = t.Kind
		case "elems":
			idx, ok := x.Lit.(lit.Indexer)
			if !ok {
				return cor.Errorf("expect indexer got %T", x.Lit)
			}
			if len(m.Elems) == 0 {
				n := idx.Len()
				m.Elems = make([]*Elem, 0, n)
				m.Type.Params = make([]typ.Param, 0, n)
			}
			err = idx.IterIdx(func(i int, el lit.Lit) error {
				return addElemFromDict(m, el.(*lit.Dict))
			})
		default:
			err = setNode(&m.Common, x)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
func (m *Model) String() string { return bfr.String(m) }
func (m *Model) WriteBfr(b *bfr.Ctx) error {
	b.WriteString("{name:")
	b.Quote(m.Name)
	b.WriteString(" type:")
	b.Quote(m.Type.Kind.String())
	b.WriteString(" elems:[")
	for i, e := range m.Elems {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteByte('{')
		if len(m.Type.Params) > 0 {
			p := m.Type.Params[i]
			if p.Name != "" {
				b.WriteString("name:")
				b.Quote(p.Name)
				b.WriteByte(' ')
			}
			b.WriteString("type:")
			if p.Kind&typ.KindCtx != 0 {
				b.Quote("~" + p.Ref)
			} else {
				b.Quote(p.Type.String())
			}
		} else if len(m.Type.Consts) > 0 {
			c := m.Type.Consts[i]
			if c.Name != "" {
				b.WriteString("name:")
				b.Quote(string(c.Name))
				b.WriteByte(' ')
			}
			b.Fmt("val:%d", c.Val)
		}
		if e.Bits != 0 {
			b.Fmt(" bits:%d", e.Bits)
		}
		if e.Ref != "" {
			b.Fmt(" ref:'%s'", e.Ref)
		}
		err := writeExtra(b, e.Extra)
		if err != nil {
			return err
		}
		b.WriteByte('}')
	}
	b.WriteByte(']')
	err := writeExtra(b, m.Extra)
	b.WriteByte('}')
	return err
}

func (s *Schema) FromDict(d *lit.Dict) (err error) {
	for _, x := range d.List {
		switch x.Key {
		case "models":
			idx, ok := x.Lit.(lit.Indexer)
			if !ok {
				return cor.Errorf("expect indexer got %T", x.Lit)
			}
			if len(s.Models) == 0 {
				s.Models = make([]*Model, 0, idx.Len())
			}
			err = idx.IterIdx(func(i int, el lit.Lit) error {
				var m Model
				m.Schema = s.Key()
				err := m.FromDict(el.(*lit.Dict))
				m.Type.Ref = m.Schema + "." + m.Name
				s.Models = append(s.Models, &m)
				return err
			})
		default:
			err = setNode(&s.Common, x)
		}
		if err != nil {
			return err
		}
	}
	return nil

}
func (s *Schema) String() string { return bfr.String(s) }
func (s *Schema) WriteBfr(b *bfr.Ctx) error {
	b.WriteString("{name:")
	b.Quote(s.Name)
	if len(s.Models) > 0 {
		b.WriteString(" models:[")
		for i, m := range s.Models {
			if i > 0 {
				b.WriteByte(' ')
			}
			err := m.WriteBfr(b)
			if err != nil {
				return err
			}
		}
		b.WriteByte(']')
	}
	err := writeExtra(b, s.Extra)
	b.WriteByte('}')
	return err
}

func (p *Project) FromDict(d *lit.Dict) (err error) {
	for _, x := range d.List {
		switch x.Key {
		case "schemas":
			idx, ok := x.Lit.(lit.Indexer)
			if !ok {
				return cor.Errorf("expect indexer got %T", x.Lit)
			}
			if len(p.Schemas) == 0 {
				p.Schemas = make([]*Schema, 0, idx.Len())
			}
			err = idx.IterIdx(func(i int, el lit.Lit) error {
				var s Schema
				err := s.FromDict(el.(*lit.Dict))
				p.Schemas = append(p.Schemas, &s)
				return err
			})
		default:
			err = setNode(&p.Common, x)
		}
		if err != nil {
			return err
		}
	}
	return nil

}
func (p *Project) String() string { return bfr.String(p) }
func (p *Project) WriteBfr(b *bfr.Ctx) error {
	b.WriteString("{name:")
	b.Quote(p.Name)
	b.WriteString(" schemas:[")
	for i, s := range p.Schemas {
		if i > 0 {
			b.WriteByte(' ')
		}
		err := s.WriteBfr(b)
		if err != nil {
			return err
		}
	}
	b.WriteString("]")
	err := writeExtra(b, p.Extra)
	b.WriteByte('}')
	return err
}

func writeExtra(b *bfr.Ctx, extra *lit.Dict) (err error) {
	if extra != nil && len(extra.List) > 0 {
		for _, x := range extra.List {
			b.WriteByte(' ')
			b.WriteString(x.Key)
			b.WriteByte(':')
			if x.Lit.Typ().Kind&typ.KindAny != 0 {
				err = x.Lit.WriteBfr(b)
			} else {
				err = b.Quote(x.Lit.String())
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}
