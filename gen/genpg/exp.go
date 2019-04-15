package genpg

import (
	"strings"

	"github.com/mb0/daql/gen"
	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

type (
	writeRaw struct {
		raw  string
		prec int
	}
	writeFunc  func(*gen.Ctx, exp.Env, *exp.Expr) error
	writeArith struct {
		op   string
		prec int
	}
	writeCmp   string
	writeLogic struct {
		op   string
		not  bool
		prec int
	}
	writeEq struct {
		op     string
		strict bool
	}
)

func (r writeRaw) WriteExpr(b *gen.Ctx, env exp.Env, e *exp.Expr) error {
	restore := b.Prec(r.prec)
	b.WriteString(r.raw)
	restore()
	return nil
}
func (r writeFunc) WriteExpr(b *gen.Ctx, env exp.Env, e *exp.Expr) error { return r(b, env, e) }
func (r writeLogic) WriteExpr(b *gen.Ctx, env exp.Env, e *exp.Expr) error {
	restore := b.Prec(r.prec)
	for i, arg := range e.Args {
		if i > 0 {
			b.WriteString(r.op)
		}
		err := writeBool(b, env, r.not, arg)
		if err != nil {
			return err
		}
	}
	restore()
	return nil
}

func (r writeArith) WriteExpr(b *gen.Ctx, env exp.Env, e *exp.Expr) error {
	restore := b.Prec(r.prec)
	for i, arg := range e.Args {
		if i > 0 {
			b.WriteString(r.op)
		}
		err := WriteEl(b, env, arg)
		if err != nil {
			return err
		}
	}
	restore()
	return nil
}

func renderIf(b *gen.Ctx, env exp.Env, e *exp.Expr) error {
	restore := b.Prec(PrecDef)
	b.WriteString("CASE ")
	var i int
	for i = 0; i+1 < len(e.Args); i += 2 {
		b.WriteString("WHEN ")
		err := writeBool(b, env, false, e.Args[i])
		if err != nil {
			return err
		}
		b.WriteString(" THEN ")
		err = WriteEl(b, env, e.Args[i+1])
		if err != nil {
			return err
		}
	}
	if i < len(e.Args) {
		b.WriteString(" ELSE ")
		err := WriteEl(b, env, e.Args[i])
		if err != nil {
			return err
		}
	}
	b.WriteString(" END")
	restore()
	return nil
}

func (r writeEq) WriteExpr(b *gen.Ctx, env exp.Env, e *exp.Expr) error {
	if len(e.Args) > 2 {
		defer b.Prec(PrecAnd)()
	}
	// TODO mind nulls
	fst, err := writeString(b, env, e.Args[0])
	if err != nil {
		return err
	}
	for i, arg := range e.Args[1:] {
		if i > 0 {
			b.WriteString(" AND ")
		}
		if !r.strict {
			restore := b.Prec(PrecCmp)
			b.WriteString(fst)
			b.WriteString(r.op)
			err = WriteEl(b, env, arg)
			if err != nil {
				return err
			}
			restore()
			continue
		}
		b.WriteByte('(')
		b.WriteString(fst)
		b.WriteString(r.op)
		oth, err := writeString(b, env, arg)
		if err != nil {
			return err
		}
		b.WriteString(oth)
		b.WriteString(" AND pg_typeof(")
		b.WriteString(fst)
		b.WriteByte(')')
		b.WriteString(r.op)
		b.WriteString("pg_typeof(")
		b.WriteString(oth)
		b.WriteByte(')')
		b.WriteByte(')')
	}
	return nil
}

func (r writeCmp) WriteExpr(b *gen.Ctx, env exp.Env, e *exp.Expr) error {
	if len(e.Args) > 2 {
		defer b.Prec(PrecAnd)()
	}
	// TODO mind nulls
	last, err := writeString(b, env, e.Args[0])
	if err != nil {
		return err
	}
	for i, arg := range e.Args[1:] {
		if i > 0 {
			b.WriteString(" AND ")
		}
		restore := b.Prec(PrecCmp)
		b.WriteString(last)
		b.WriteString(string(r))
		oth, err := writeString(b, env, arg)
		if err != nil {
			return err
		}
		b.WriteString(oth)
		restore()
		last = oth
	}
	return nil
}

func writeAs(b *gen.Ctx, env exp.Env, e *exp.Expr) error {
	if len(e.Args) == 0 {
		return cor.Errorf("empty as expression")
	}
	t, ok := e.Args[0].(typ.Type)
	if !ok {
		return cor.Errorf("as expression must start with a type")
	}
	ts, err := TypString(t)
	if err != nil {
		return err
	}
	switch len(e.Args) {
	case 1:
		zero, _, err := zeroStrings(t)
		if err != nil {
			return err
		}
		b.WriteString(zero)
	case 2:
		err = WriteEl(b, env, e.Args[1])
		if err != nil {
			return err
		}
	default:
		return cor.Errorf("not implemented")
	}
	b.WriteString("::")
	b.WriteString(ts)
	return nil
}

func writeCat(b *gen.Ctx, env exp.Env, e *exp.Expr) error {
	restore := b.Prec(PrecDef)
	for i, arg := range e.Args {
		if i > 0 {
			b.WriteString(" || ")
		}
		// TODO cast to element type
		err := WriteEl(b, env, arg)
		if err != nil {
			return err
		}
	}
	restore()
	return nil
}
func writeApd(b *gen.Ctx, env exp.Env, e *exp.Expr) error {
	if len(e.Args) == 0 {
		return cor.Errorf("empty apd expression")
	}
	t := elType(e.Args[0])
	if t == typ.Void {
		return cor.Errorf("untyped first argument in apd expression")
	}
	restore := b.Prec(PrecDef)
	// either jsonb or postgres array
	ispg := t.Elem().Kind&typ.MaskPrim != 0
	for i, arg := range e.Args {
		if i > 0 {
			b.WriteString(" || ")
		}
		// TODO cast to element type when ispg, otherwise jsonb
		_ = ispg
		err := WriteEl(b, env, arg)
		if err != nil {
			return err
		}
	}
	restore()
	return nil
}

var layoutSet = []typ.Param{{Name: "a", Type: typ.Dict}, {Name: "unis"}}

func writeSet(b *gen.Ctx, env exp.Env, e *exp.Expr) error {
	// First arg can only be a jsonb obj
	// TODO but check that
	lo, err := exp.LayoutArgs(layoutSet, e.Args)
	if err != nil {
		return err
	}
	decls, err := lo.Unis(1)
	if err != nil {
		return err
	}
	// Collect literals and other decls. We can merge all literal directly,
	// but need to use jsonb_set for references and other expressions.
	dict := &lit.Dict{}
	var rest []exp.Decl
	for _, d := range decls {
		switch v := d.Args[0].(type) {
		case lit.Lit:
			dict.SetKey(d.Name, v)
		case *exp.Sym:
			rest = append(rest, d)
		case *exp.Expr:
			rest = append(rest, d)
		}
	}
	for range rest {
		b.WriteString("jsonb_set(")
	}
	err = WriteEl(b, env, e.Args[0])
	if err != nil {
		return err
	}
	if dict.Len() > 0 {
		b.WriteString(" || ")
		err = WriteLit(b, dict)
		if err != nil {
			return err
		}
	}
	for i := range rest {
		d := rest[len(rest)-i-1]
		b.WriteString(", {")
		b.WriteString(strings.ToLower(d.Name))
		b.WriteString("}, ")
		err = WriteEl(b, env, d.Args[0])
		if err != nil {
			return err
		}
		b.WriteString(", true)")
	}
	return nil
}

func writeBool(b *gen.Ctx, env exp.Env, not bool, e exp.El) error {
	var t typ.Type
	switch v := e.(type) {
	case lit.Lit:
		t = v.Typ()
	case *exp.Sym:
		t = v.Type
	case *exp.Expr:
		t = v.Rslv.Res()
	default:
		return cor.Errorf("unexpected element %s", e)
	}
	if t.Kind == typ.KindBool {
		if not {
			defer b.Prec(PrecNot)()
			b.WriteString("NOT ")
		}
		return WriteEl(b, env, e)
	}
	// add boolean conversion if necessary
	if t.Kind&typ.FlagOpt != 0 {
		defer b.Prec(PrecIs)()
		err := WriteEl(b, env, e)
		if err != nil {
			return err
		}
		if not {
			b.WriteString(" IS NULL")
		} else {
			b.WriteString(" IS NOT NULL")
		}
		return nil
	}
	cmp, oth, err := zeroStrings(t)
	if err != nil {
		return err
	}
	if oth != "" {
		if not {
			defer b.Prec(PrecOr)()
		} else {
			defer b.Prec(PrecAnd)()
		}
	} else if cmp != "" {
		defer b.Prec(PrecCmp)()
	}
	err = WriteEl(b, env, e)
	if err != nil {
		return err
	}
	if cmp != "" {
		op := " != "
		if not {
			op = " = "
		}
		restore := b.Prec(PrecCmp)
		b.WriteString(op)
		b.WriteString(cmp)
		if oth != "" {
			if not {
				b.WriteString(" OR ")
			} else {
				b.WriteString(" AND ")
			}
			err := WriteEl(b, env, e)
			if err != nil {
				return err
			}
			b.WriteString(op)
			b.WriteString(oth)
		}
		restore()
	}
	return nil
}

func writeString(c *gen.Ctx, env exp.Env, e exp.El) (string, error) {
	cc := *c
	var b strings.Builder
	cc.Ctx = bfr.Ctx{B: &b}
	err := WriteEl(&cc, env, e)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func elType(e exp.El) typ.Type {
	switch v := e.(type) {
	case lit.Lit:
		return v.Typ()
	case *exp.Sym:
		return v.Type
	case *exp.Expr:
		return v.Rslv.Res()
	}
	return typ.Void
}

func zeroStrings(t typ.Type) (zero, alt string, _ error) {
	switch t.Kind & typ.SlotMask {
	case typ.KindBool:
	case typ.BaseNum, typ.KindInt, typ.KindReal, typ.KindFlag:
		zero = "0"
	case typ.BaseChar, typ.KindStr, typ.KindRaw:
		zero = "''"
	case typ.KindSpan:
		zero = "'0'"
	case typ.KindTime:
		zero = "'0001-01-01Z'"
	case typ.KindEnum:
		// TODO
	case typ.KindArr:
		// TODO check if postgres array otherwise
		fallthrough
	case typ.BaseList:
		zero, alt = "'null'", "'[]'"
	case typ.BaseDict, typ.KindObj, typ.KindRec:
		zero, alt = "'null'", "'{}'"
	default:
		return "", "", cor.Errorf("error unexpected type %s", t)
	}
	return
}
