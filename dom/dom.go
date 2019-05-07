package dom

import (
	"strings"

	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

// Bit is a flag used for a number of field options.
type Bit uint64

const (
	BitOpt = 1 << iota
	BitPK
	BitIdx
	BitUniq
	BitOrdr
	BitAuto
	BitRO
)

func (Bit) Flags() map[string]int64 { return bitConsts }

// Keys is a slice of field keys used for indices and order by definitions.
type Keys []string

// Elem holds additional information for either constants or type paramters.
type Elem struct {
	Bits  Bit       `json:"bits,omitempty"`
	Extra *lit.Dict `json:"extra,omitempty"`
}

// Index represents a record model index, mainly used for databases.
type Index struct {
	Name   string `json:"name,omitempty"`
	Keys   Keys   `json:"keys"`
	Unique bool   `json:"unique,omitempty"`
}

// Node represents the common name and version of a model, schema or project.
type Node struct {
	Vers  int64     `json:"vers,omitempty"`
	Extra *lit.Dict `json:"extra,omitempty"`
	Name  string    `json:"name,omitempty"`
	key   string
}

func (n *Node) Key() string {
	if n.key == "" && n.Name != "" {
		n.key = strings.ToLower(n.Name)
	}
	return n.key
}

// Model represents either a flag, enum or record type and has extra domain information.
type Model struct {
	Node
	typ.Type `json:"typ"`
	Elems    []*Elem `json:"elems,omitempty"`
	Rec      *Record `json:"rec,omitempty"`
	schema   string
}

// Record holds data specific to record types for grouping.
type Record struct {
	Indices []*Index `json:"indices,omitempty"`
	OrderBy Keys     `json:"orderby,omitempty"`
	// TODO add triggers and references

}

func (m *Model) Qual() string {
	return m.schema
}

// Schema is a namespace for models.
type Schema struct {
	Node
	Path   string   `json:"path,omitempty"`
	Use    Keys     `json:"use,omitempty"`
	Models []*Model `json:"models"`
}

// Project is a collection of schemas.
type Project struct {
	Node
	Schemas []*Schema `json:"schemas"`
}

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
		for i, c := range m.Consts {
			if c.Key() == key {
				return ConstElem{&m.Consts[i], m.Elems[i]}
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
		_, i, err := m.ParamByKey(key)
		if err == nil {
			return FieldElem{&m.Params[i], m.Elems[i]}
		}
	}
	return FieldElem{}
}

var bitConsts = map[string]int64{
	"Opt":  BitOpt,
	"PK":   BitPK,
	"Idx":  BitIdx,
	"Uniq": BitUniq,
	"Ordr": BitOrdr,
	"Auto": BitAuto,
	"RO":   BitRO,
}

func setNode(n *Node, x lit.Keyed) error {
	switch x.Key {
	case "name":
		n.Name = x.Lit.(lit.Character).Char()
	case "vers":
		n.Vers = int64(x.Lit.(lit.Numeric).Num())
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
		case "typ":
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
	if m.Kind&typ.KindPrim != 0 {
		m.Consts = append(m.Consts, c)
	} else {
		m.Params = append(m.Params, p)
	}
	m.Elems = append(m.Elems, &el)
	return nil
}

func (m *Model) FromDict(d *lit.Dict) (err error) {
	if m.Info == nil {
		m.Type.Kind = typ.KindObj
		m.Info = &typ.Info{}
	}
	for _, x := range d.List {
		switch x.Key {
		case "typ":
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
				m.Params = make([]typ.Param, 0, n)
			}
			err = idx.IterIdx(func(i int, el lit.Lit) error {
				return addElemFromDict(m, el.(*lit.Dict))
			})
		default:
			err = setNode(&m.Node, x)
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
	b.WriteString(" typ:")
	b.Quote(m.Kind.String())
	b.WriteString(" elems:[")
	for i, e := range m.Elems {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteByte('{')
		if len(m.Params) > 0 {
			p := m.Params[i]
			if p.Name != "" {
				b.WriteString("name:")
				b.Quote(p.Name)
				b.WriteByte(' ')
			}
			b.WriteString("typ:")
			if p.HasRef() {
				b.Quote("@" + p.Ref)
			} else {
				b.Quote(p.Type.String())
			}
		} else if len(m.Consts) > 0 {
			c := m.Consts[i]
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
				m.schema = s.Key()
				err := m.FromDict(el.(*lit.Dict))
				m.Ref = m.schema + "." + m.Name
				s.Models = append(s.Models, &m)
				return err
			})
		default:
			err = setNode(&s.Node, x)
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
			err = setNode(&p.Node, x)
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
	if p.Vers != 0 {
		b.Fmt(" vers:%d", p.Vers)
	}
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

func writeExtra(b *bfr.Ctx, extra *lit.Dict) error {
	if extra != nil && len(extra.List) > 0 {
		for _, x := range extra.List {
			b.WriteByte(' ')
			b.WriteString(x.Key)
			b.WriteByte(':')
			err := x.Lit.WriteBfr(b)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
