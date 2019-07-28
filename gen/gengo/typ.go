package gengo

import (
	"strings"

	"github.com/mb0/daql/gen"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/typ"
	"github.com/pkg/errors"
)

// WriteType writes the native go type for t to c or returns an error.
func WriteType(c *gen.Gen, t typ.Type) error {
	k := t.Kind
	switch k {
	case typ.KindAny:
		return c.Fmt(Import(c, "lit.Lit"))
	case typ.KindDyn:
		return c.Fmt(Import(c, "*exp.Dyn"))
	case typ.KindTyp:
		return c.Fmt(Import(c, "typ.Type"))
	case typ.KindExpr:
		return c.Fmt(Import(c, "exp.El"))
	}
	var r string
	switch k & typ.MaskRef {
	case typ.KindNum:
		r = "float64"
	case typ.KindBool:
		r = "bool"
	case typ.KindInt:
		r = "int64"
	case typ.KindReal:
		r = "float64"
	case typ.KindChar, typ.KindStr:
		r = "string"
	case typ.KindRaw:
		r = "[]byte"
	case typ.KindUUID:
		r = "[16]byte"
	case typ.KindTime:
		r = Import(c, "time.Time")
	case typ.KindSpan:
		r = Import(c, "time.Duration")
	case typ.KindIdxr:
		return c.Fmt(Import(c, "lit.List"))
	case typ.KindList:
		c.WriteString("[]")
		return WriteType(c, t.Elem())
	case typ.KindKeyr:
		return c.Fmt(Import(c, "*lit.Dict"))
	case typ.KindDict:
		//c.WriteString("map[string]")
		//return WriteType(c, t.Elem())
		return c.Fmt(Import(c, "*lit.Dict"))
	case typ.KindRec:
		if k&typ.KindOpt != 0 {
			c.WriteByte('*')
		}
		c.WriteString("struct {\n")
		if !t.HasParams() {
			return typ.ErrInvalid
		}
		for _, f := range t.Info.Params {
			name, opt := f.Name, f.Opt()
			if opt {
				name = name[:len(name)-1]
			}
			c.WriteByte('\t')
			if name != "" {
				c.WriteString(name)
				c.WriteByte(' ')
			}
			err := WriteType(c, f.Type)
			if err != nil {
				return cor.Errorf("write field %s: %w", f.Name, err)
			}
			if name != "" {
				c.WriteString(" `json:\"")
				c.WriteString(strings.ToLower(name))
				if opt {
					c.WriteString(",omitempty")
				}
				c.WriteString("\"`")
			}
			c.WriteByte('\n')
		}
		c.WriteByte('}')
		return nil
	case typ.KindBits, typ.KindEnum, typ.KindObj:
		r = Import(c, refName(t))
	}
	if r == "" {
		return errors.Errorf("type %s %s cannot be represented in go", t, t.Kind)
	}
	if k&typ.KindOpt != 0 {
		c.WriteByte('*')
	}
	c.WriteString(r)
	return nil
}
