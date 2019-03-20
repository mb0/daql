package genpg

import (
	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/bfr"
)

func WriteEnum(b *bfr.Ctx, m *dom.Model) error {
	b.WriteString("CREATE TYPE ")
	b.WriteString(m.Ref())
	b.WriteString(" AS ENUM (")
	for i, c := range m.Consts {
		if i > 0 {
			b.WriteString(", ")
		}
		WriteQuote(b, c.Name)
	}
	return b.WriteByte(')')
}

func WriteTable(b *bfr.Ctx, m *dom.Model) error {
	b.WriteString("CREATE TABLE ")
	b.WriteString(m.Ref())
	b.WriteString(" (")
	for i, f := range m.Fields {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(f.Key())
		b.WriteByte(' ')
		ts, err := TypString(f.Type)
		if err != nil {
			return err
		}
		b.WriteString(ts)
		if f.Bits&dom.BitPK != 0 {
			b.WriteString(" PRIMARY KEY")
			// TODO auto
		} else if f.Type.IsOpt() {
			b.WriteString(" NULL")
		} else {
			b.WriteString(" NOT NULL")
			if f.Bits&dom.BitOpt != 0 { // && f.Def == nil
				// TODO implicit default
			}
		}
		// TODO default
		// TODO references
	}
	return b.WriteByte(')')
}
