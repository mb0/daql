package dom

import (
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/typ"
)

// Keys is a slice of field keys used for indices and order by definitions.
type Keys []string

type Extra map[string]interface{}

// Display holds optional display information for its declaration.
//
// All fields should either be localized version for a single target language or a translation key,
// that is then used with a separate localization system.
type Display struct {
	Label string `json:"label,omitempty"`
	Descr string `json:"descr,omitempty"`
	Help  string `json:"help,omitempty"`
	Doc   string `json:"doc,omitempty"`
	Fmt   string `json:"fmt,omitempty"`
}

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

// Field represents a record field with extra domain information.
type Field struct {
	Name  string   `json:"name,omitempty"`
	Type  typ.Type `json:"typ"`
	Bits  Bit      `json:"bits"`
	Extra Extra    `json:"extra,omitempty"`
	*Display
	key string
}

func (f *Field) Key() string {
	if f.key == "" && f.Name != "" {
		f.key = strings.ToLower(f.Name)
	}
	return f.key
}

// Index represents a record model index, mainly used for databases.
type Index struct {
	Name   string `json:"name,omitempty"`
	Keys   Keys   `json:"keys"`
	Unique bool   `json:"unique,omitempty"`
}

// Model represents either a flag, enum or record type and has extra domain information.
type Model struct {
	Name string   `json:"name"`
	Kind typ.Kind `json:"kind"`
	Display
	Fields   []*Field    `json:"fields,omitempty"`
	Indices  []*Index    `json:"indices,omitempty"`
	Consts   []cor.Const `json:"consts,omitemtpy"`
	OrderBy  Keys        `json:"orderby,omitempty"`
	Extra    Extra       `json:"extra,omitempty"`
	schema   string
	ref, key string
	typ      typ.Type
}

func (m *Model) Ref() string {
	if m.ref == "" {
		m.ref = m.schema + "." + m.Name
	}
	return m.ref
}

func (m *Model) Key() string {
	if m.key == "" {
		m.key = strings.ToLower(m.Name)
	}
	return m.key
}

func (m *Model) Typ() typ.Type {
	if m.typ == typ.Void {
		fs := make([]typ.Field, 0, len(m.Fields))
		for _, f := range m.Fields {
			name := f.Name
			if f.Bits&BitOpt != 0 {
				name += "?"
			}
			fs = append(fs, typ.Field{Name: name, Type: f.Type})
		}
		m.typ = typ.Rec(m.Ref())
		m.typ.Fields = fs
	}
	return m.typ
}

// Schema is a namespace for models.
type Schema struct {
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
	Display
	Use    Keys     `json:"use,omitempty"`
	Models []*Model `json:"models"`
	Extra  Extra    `json:"extra,omitempty"`
}

// Project is a collection of schemas.
type Project struct {
	Name string `json:"name"`
	Display
	Schemas []*Schema `json:"schemas"`
	Extra   Extra     `json:"extra,omitempty"`
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
