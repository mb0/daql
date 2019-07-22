// generated code

package dom

import (
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

// Bit is a bit set used for a number of field options.
type Bit uint64

const (
	BitOpt Bit = 1 << iota
	BitPK
	BitIdx
	BitUniq
	BitOrdr
	BitAuto
	BitRO
)

// Elem holds additional information for either constants or type paramters.
type Elem struct {
	Bits  Bit       `json:"bits,omitempty"`
	Extra *lit.Dict `json:"extra,omitempty"`
	Ref   string    `json:"ref,omitempty"`
}

// Index represents a record model index, mainly used for databases.
type Index struct {
	Name   string   `json:"name,omitempty"`
	Keys   []string `json:"keys"`
	Unique bool     `json:"unique,omitempty"`
}

// Common represents the common name and version of model, schema or project nodes.
type Common struct {
	Name  string    `json:"name,omitempty"`
	Extra *lit.Dict `json:"extra,omitempty"`
}

// Object holds data specific to object types for grouping.
type Object struct {
	Indices []*Index `json:"indices,omitempty"`
	OrderBy []string `json:"orderby,omitempty"`
}

// Model represents either a bits, enum or record type and has extra domain information.
type Model struct {
	Common
	Type   typ.Type `json:"type"`
	Elems  []*Elem  `json:"elems,omitempty"`
	Object *Object  `json:"object,omitempty"`
	Schema string   `json:"schema,omitempty"`
}

// Schema is a namespace for models.
type Schema struct {
	Common
	Path   string   `json:"path,omitempty"`
	Use    []string `json:"use,omitempty"`
	Models []*Model `json:"models"`
}

// Project is a collection of schemas and is the central place for any extra project configuration.
//
// The schema definition can either be declared as part of the project file, or included from an
// external schema file. Includes should have syntax to filtering the included schema definition.
//
// Extra setting, usually include, but are not limited to, targets and output paths for code
// generation, paths to look for the project's manifest and history.
type Project struct {
	Common
	Schemas []*Schema `json:"schemas"`
}
