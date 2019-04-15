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
	case typ.KindInt, typ.KindFlag:
		return "int8", nil
	case typ.BaseNum, typ.KindReal:
		return "float8", nil
	case typ.BaseChar, typ.KindStr:
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
	case typ.KindAny, typ.BaseList, typ.BaseDict, typ.KindMap, typ.KindObj, typ.KindRec:
		return "jsonb", nil
	case typ.KindArr:
		if n := t.Elem(); n.Kind&typ.MaskPrim != 0 {
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
	if t.Kind&typ.FlagOpt != 0 && l.IsZero() {
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
	case typ.BaseNum, typ.KindInt, typ.KindReal, typ.KindFlag:
		return l.WriteBfr(&b.Ctx)
	case typ.BaseChar, typ.KindStr:
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
	case typ.BaseList:
		return writeJSONB(b, l)
	case typ.KindArr:
		if t.Elem().Kind&typ.MaskPrim != 0 {
			// use postgres array for one dimensional primitive arrays
			return writeArray(b, l.(lit.Arr))
		}
		return writeJSONB(b, l) // otherwise use jsonb
	case typ.BaseDict, typ.KindMap, typ.KindObj, typ.KindRec:
		return writeJSONB(b, l)
	}
	return nil
}
