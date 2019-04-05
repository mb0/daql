package genpg

import (
	"io/ioutil"
	"strings"

	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/gen"
	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/typ"
)

func WriteFile(c *gen.Ctx, fname string, s *dom.Schema) error {
	b := bfr.Get()
	defer bfr.Put(b)
	c.Ctx = bfr.Ctx{B: b, Tab: "\t"}
	err := RenderFile(c, s)
	if err != nil {
		return cor.Errorf("render file %s error: %v", fname, err)
	}
	err = ioutil.WriteFile(fname, b.Bytes(), 0644)
	if err != nil {
		return cor.Errorf("write file %s error: %v", fname, err)
	}
	return nil
}

func RenderFile(c *gen.Ctx, s *dom.Schema) (err error) {
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
	b.WriteString(m.Type.Key())
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
	b.WriteString(m.Type.Key())
	b.WriteString(" (")
	b.Indent()
	for i, p := range m.Params {
		if i > 0 {
			b.WriteByte(',')
			if !b.Break() {
				b.WriteByte(' ')
			}
		}
		writeField(b, p, m.Elems[i])
	}
	b.Dedent()
	return b.WriteByte(')')
}

func writeField(b *gen.Ctx, p typ.Param, el *dom.Elem) error {
	key := p.Key()
	if key == "" {
		switch p.Type.Kind & typ.MaskRef {
		case typ.KindFlag, typ.KindEnum:
			split := strings.Split(p.Type.Key(), ".")
			key = split[len(split)-1]
		case typ.KindRec:
			return embedField(b, p.Type)
		default:
			return cor.Errorf("unexpected embedded field type %s", p.Type)
		}
	}
	b.WriteString(key)
	b.WriteByte(' ')
	ts, err := TypString(p.Type)
	if err != nil {
		return err
	}
	if ts == "int8" && el.Bits&dom.BitPK != 0 && el.Bits&dom.BitAuto != 0 {
		b.WriteString("serial8")
	} else {
		b.WriteString(ts)
	}
	if el.Bits&dom.BitPK != 0 {
		b.WriteString(" PRIMARY KEY")
		// TODO auto
	} else if el.Bits&dom.BitOpt != 0 || p.Type.IsOpt() {
		b.WriteString(" NULL")
	} else {
		b.WriteString(" NOT NULL")
	}
	// TODO default
	// TODO references
	return nil
}

func embedField(b *gen.Ctx, t typ.Type) error {
	split := strings.Split(t.Key(), ".")
	m := b.Project.Schema(split[0]).Model(split[1])
	for i, p := range m.Params {
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
