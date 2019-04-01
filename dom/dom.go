package dom

import (
	"strings"

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

func (Bit) Flag() []cor.Const { return bitConsts }

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
	Vers int64  `json:"vers,omitempty"`
	Name string `json:"name,omitempty"`
	key  string
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
	typ.Type
	Elems  []*Elem   `json:"elems,omitempty"`
	Rec    *Record   `json:"rec,omitempty"`
	Extra  *lit.Dict `json:"extra,omitempty"`
	schema string
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
	Path   string    `json:"path,omitempty"`
	Use    Keys      `json:"use,omitempty"`
	Models []*Model  `json:"models"`
	Extra  *lit.Dict `json:"extra,omitempty"`
}

// Project is a collection of schemas.
type Project struct {
	Node
	Schemas []*Schema `json:"schemas"`
	Extra   *lit.Dict `json:"extra,omitempty"`
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
	*cor.Const
	*Elem
}

// Const returns a constant element for key or nil.
func (m *Model) Const(key string) ConstElem {
	if m != nil {
		for i, c := range m.Consts {
			if strings.EqualFold(c.Name, key) {
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

var bitConsts = []cor.Const{
	{"Opt", BitOpt},
	{"PK", BitPK},
	{"Idx", BitIdx},
	{"Uniq", BitUniq},
	{"Ordr", BitOrdr},
	{"Auto", BitAuto},
	{"RO", BitRO},
}
