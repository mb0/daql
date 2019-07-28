package genpg

import (
	"io"
	"os"
	"strings"

	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/typ"
)

func WriteFile(fname string, p *dom.Project, s *dom.Schema) error {
	b := bfr.Get()
	defer bfr.Put(b)
	w := NewWriter(b, ExpEnv{})
	w.Project = p
	w.WriteString(w.Header)
	w.WriteString("BEGIN;\n\n")
	err := w.WriteSchema(s)
	if err != nil {
		return cor.Errorf("render file %s error: %v", fname, err)
	}
	w.WriteString("COMMIT;\n")
	f, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, b)
	return err
}

func (w *Writer) WriteSchema(s *dom.Schema) (err error) {
	w.WriteString("CREATE SCHEMA ")
	w.WriteString(s.Name)
	w.WriteString(";\n\n")
	for _, m := range s.Models {
		switch m.Type.Kind {
		case typ.KindBits:
		case typ.KindEnum:
			err = w.WriteEnum(m)
		default:
			err = w.WriteTable(m)
		}
		if err != nil {
			return err
		}
		w.WriteString(";\n\n")
	}
	return nil
}

func (w *Writer) WriteEnum(m *dom.Model) error {
	w.WriteString("CREATE TYPE ")
	w.WriteString(m.Type.Key())
	w.WriteString(" AS ENUM (")
	w.Indent()
	for i, c := range m.Type.Consts {
		if i > 0 {
			w.WriteByte(',')
			if !w.Break() {
				w.WriteByte(' ')
			}
		}
		WriteQuote(w, c.Key())
	}
	w.Dedent()
	return w.WriteByte(')')
}

func (w *Writer) WriteTable(m *dom.Model) error {
	w.WriteString("CREATE TABLE ")
	w.WriteString(m.Type.Key())
	w.WriteString(" (")
	w.Indent()
	for i, p := range m.Type.Params {
		if i > 0 {
			w.WriteByte(',')
			if !w.Break() {
				w.WriteByte(' ')
			}
		}
		err := w.writeField(p, m.Elems[i])
		if err != nil {
			return err
		}
	}
	w.Dedent()
	return w.WriteByte(')')
}

func (w *Writer) writeField(p typ.Param, el *dom.Elem) error {
	key := p.Key()
	if key == "" {
		switch p.Type.Kind & typ.MaskRef {
		case typ.KindBits, typ.KindEnum:
			split := strings.Split(p.Type.Key(), ".")
			key = split[len(split)-1]
		case typ.KindObj:
			return w.writerEmbed(p.Type)
		default:
			return cor.Errorf("unexpected embedded field type %s", p.Type)
		}
	}
	w.WriteString(key)
	w.WriteByte(' ')
	ts, err := TypString(p.Type)
	if err != nil {
		return err
	}
	if ts == "int8" && el.Bits&dom.BitPK != 0 && el.Bits&dom.BitAuto != 0 {
		w.WriteString("serial8")
	} else {
		w.WriteString(ts)
	}
	if el.Bits&dom.BitPK != 0 {
		w.WriteString(" PRIMARY KEY")
		// TODO auto
	} else if el.Bits&dom.BitOpt != 0 || p.Type.IsOpt() {
		w.WriteString(" NULL")
	} else {
		w.WriteString(" NOT NULL")
	}
	// TODO default
	// TODO references
	return nil
}

func (w *Writer) writerEmbed(t typ.Type) error {
	split := strings.Split(t.Key(), ".")
	m := w.Project.Schema(split[0]).Model(split[1])
	if m == nil {
		return cor.Errorf("no model for %s in %s", t.Key(), m)
	}
	for i, p := range m.Type.Params {
		if i > 0 {
			w.WriteByte(',')
			if !w.Break() {
				w.WriteByte(' ')
			}
		}
		key := p.Key()
		if key == "" {
			w.writerEmbed(p.Type)
			continue
		}
		w.WriteString(p.Key())
		w.WriteByte(' ')
		ts, err := TypString(p.Type)
		if err != nil {
			return err
		}
		w.WriteString(ts)
		if p.Opt() || p.Type.IsOpt() {
			w.WriteString(" NULL")
		} else {
			w.WriteString(" NOT NULL")
			if p.Opt() {
				// TODO implicit default
			}
		}
	}
	return nil
}
