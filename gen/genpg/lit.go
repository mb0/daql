package genpg

import (
	"github.com/mb0/daql/gen"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

func TypString(t typ.Type) (string, error) {
	switch t.Kind & typ.MaskRef {
	case typ.KindBool:
		return "bool", nil
	case typ.KindInt, typ.KindBits:
		return "int8", nil
	case typ.KindNum, typ.KindReal:
		return "float8", nil
	case typ.KindChar, typ.KindStr:
		return "text", nil
	case typ.KindEnum:
		// TODO
		// write qualified enum name
	case typ.KindRaw:
		return "bytea", nil
	case typ.KindUUID:
		return "uuid", nil
	case typ.KindTime:
		return "timestamptz", nil
	case typ.KindSpan:
		return "interval", nil
	case typ.KindAny, typ.KindDict, typ.KindRec, typ.KindObj:
		return "jsonb", nil
	case typ.KindList:
		if n := t.Elem(); n != typ.Any && n.Kind&typ.KindPrim != 0 {
			res, err := TypString(n)
			if err != nil {
				return "", err
			}
			return res + "[]", nil
		}
		return "jsonb", nil
	}
	return "", cor.Errorf("unexpected type %s", t)
}

// WriteLit renders the literal l to b or returns an error.
func WriteLit(b *gen.Ctx, l lit.Lit) error {
	t := l.Typ()
	if (t.Kind == typ.KindAny || t.Kind&typ.KindOpt != 0) && l.IsZero() {
		return b.Fmt("NULL")
	}
	if o, ok := l.(lit.Opter); ok {
		l = o.Some()
	}
	switch t.Kind & typ.MaskRef {
	case typ.KindAny:
		return writeJSONB(b, l)
	case typ.KindBool:
		if l.IsZero() {
			return b.Fmt("FALSE")
		}
		return b.Fmt("TRUE")
	case typ.KindNum, typ.KindInt, typ.KindReal, typ.KindBits:
		return l.WriteBfr(&b.Ctx)
	case typ.KindChar, typ.KindStr:
		return l.WriteBfr(&b.Ctx)
	case typ.KindEnum:
		// TODO write string and cast with qualified enum name
	case typ.KindRaw:
		return writeSuffix(b, l, "::bytea")
	case typ.KindUUID:
		return writeSuffix(b, l, "::uuid")
	case typ.KindTime:
		return writeSuffix(b, l, "::timestamptz")
	case typ.KindSpan:
		return writeSuffix(b, l, "::interval")
	case typ.KindList:
		if k := t.Elem().Kind; k != typ.KindAny && k&typ.KindPrim != 0 {
			// use postgres array for one dimensional primitive arrays
			return writeArray(b, l.(lit.Appender))
		}
		return writeJSONB(b, l) // otherwise use jsonb
	case typ.KindDict, typ.KindRec, typ.KindObj:
		return writeJSONB(b, l)
	}
	return cor.Errorf("unexpected lit %s", l)
}
