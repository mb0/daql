package genpg

import (
	"strings"

	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/gen"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/typ"
)

func WriteFile(c *gen.Ctx, s *dom.Schema) (err error) {
	c.WriteString(c.Header)
	c.WriteString("BEGIN;\n\n")
	c.WriteString("CREATE SCHEMA ")
	c.WriteString(s.Name)
	c.WriteString(";\n\n")
	for _, m := range s.Models {
		switch m.Kind {
		case typ.KindFlag:
		case typ.KindEnum:
			err = WriteEnum(c, m)
		default:
			err = WriteTable(c, m)
		}
		if err != nil {
			return err
		}
		c.WriteString(";\n\n")
	}
	c.WriteString("COMMIT;\n")
	return nil
}

func WriteEnum(b *gen.Ctx, m *dom.Model) error {
	b.WriteString("CREATE TYPE ")
	b.WriteString(m.Ref())
	b.WriteString(" AS ENUM (")
	b.Indent()
	for i, c := range m.Consts {
		if i > 0 {
			b.WriteByte(',')
			if !b.Break() {
				b.WriteByte(' ')
			}
		}
		WriteQuote(b, c.Name)
	}
	b.Dedent()
	return b.WriteByte(')')
}

func WriteTable(b *gen.Ctx, m *dom.Model) error {
	b.WriteString("CREATE TABLE ")
	b.WriteString(m.Ref())
	b.WriteString(" (")
	b.Indent()
	for i, f := range m.Fields {
		if i > 0 {
			b.WriteByte(',')
			if !b.Break() {
				b.WriteByte(' ')
			}
		}
		writeField(b, f)
	}
	b.Dedent()
	return b.WriteByte(')')
}

func writeField(b *gen.Ctx, f *dom.Field) error {
	key := f.Key()
	if key == "" {
		switch f.Type.Kind & typ.MaskRef {
		case typ.KindFlag, typ.KindEnum:
			split := strings.Split(f.Type.Key(), ".")
			key = split[len(split)-1]
		case typ.KindRec:
			return embedField(b, f.Type)
		default:
			return cor.Errorf("unexpected embedded field type %s", f.Type)
		}
	}
	b.WriteString(key)
	b.WriteByte(' ')
	ts, err := TypString(f.Type)
	if err != nil {
		return err
	}
	if ts == "int8" && f.Bits&dom.BitPK != 0 && f.Bits&dom.BitAuto != 0 {
		b.WriteString("serial8")
	} else {
		b.WriteString(ts)
	}
	if f.Bits&dom.BitPK != 0 {
		b.WriteString(" PRIMARY KEY")
		// TODO auto
	} else if f.Bits&dom.BitOpt != 0 || f.Type.IsOpt() {
		b.WriteString(" NULL")
	} else {
		b.WriteString(" NOT NULL")
	}
	// TODO default
	// TODO references
	return nil
}

func embedField(b *gen.Ctx, t typ.Type) error {
	// TODO query embedded dom model instead
	for i, p := range t.Params {
		if i > 0 {
			b.WriteByte(',')
			if !b.Break() {
				b.WriteByte(' ')
			}
		}
		key := p.Key()
		if key == "" {
			embedField(b, p.Type)
			continue
		}
		b.WriteString(p.Key())
		b.WriteByte(' ')
		ts, err := TypString(p.Type)
		if err != nil {
			return err
		}
		b.WriteString(ts)
		if p.Opt() || p.Type.IsOpt() {
			b.WriteString(" NULL")
		} else {
			b.WriteString(" NOT NULL")
			if p.Opt() {
				// TODO implicit default
			}
		}
	}
	return nil
}
