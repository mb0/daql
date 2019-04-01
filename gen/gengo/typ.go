package gengo

import (
	"strings"

	"github.com/mb0/daql/gen"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/typ"
	"github.com/pkg/errors"
)

// WriteType writes the native go type for t to c or returns an error.
func WriteType(c *gen.Ctx, t typ.Type) error {
	k := t.Kind
	switch k {
	case typ.KindAny:
		return c.Fmt(Import(c, "lit.Lit"))
	case typ.ExpDyn:
		return c.Fmt(Import(c, "exp.Dyn"))
	}
	var r string
	switch k & typ.MaskRef {
	case typ.BaseNum:
		r = "float64"
	case typ.KindBool:
		r = "bool"
	case typ.KindInt:
		r = "int64"
	case typ.KindReal:
		r = "float64"
	case typ.BaseChar, typ.KindStr:
		r = "string"
	case typ.KindRaw:
		r = "[]byte"
	case typ.KindUUID:
		r = "[16]byte"
	case typ.KindTime:
		r = Import(c, "time.Time")
	case typ.KindSpan:
		r = Import(c, "time.Duration")
	case typ.BaseList:
		return c.Fmt(Import(c, "lit.List"))
	case typ.KindArr:
		c.WriteString("[]")
		return WriteType(c, t.Next())
	case typ.BaseDict:
		return c.Fmt(Import(c, "*lit.Dict"))
	case typ.KindMap:
		c.WriteString("map[string]")
		return WriteType(c, t.Next())
	case typ.KindObj:
		if k&typ.FlagOpt != 0 {
			c.WriteByte('*')
		}
		if t.Info == nil {
			return typ.ErrInvalid
		}
		c.WriteString("struct {\n")
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
	case typ.KindFlag, typ.KindEnum, typ.KindRec:
		// TODO lookup cased name from dom schema
		r = Import(c, refName(t))
	}
	if r == "" {
		return errors.Errorf("type %s cannot be represented in go", t)
	}
	if k&typ.FlagOpt != 0 {
		c.WriteByte('*')
	}
	c.WriteString(r)
	return nil
}
