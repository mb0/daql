package pg

import (
	"fmt"
	"strings"

	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

type (
	renderConst string
	renderFunc  func(bfr.Ctx, exp.Env, *exp.Expr) error
	renderArith string
	renderCmp   string
	renderLogic struct {
		op  string
		not bool
	}
	renderEq struct {
		op     string
		strict bool
	}
)

func (r renderConst) Render(b bfr.Ctx, env exp.Env, e *exp.Expr) error { return b.Fmt(string(r)) }
func (r renderFunc) Render(b bfr.Ctx, env exp.Env, e *exp.Expr) error  { return r(b, env, e) }
func (r renderLogic) Render(b bfr.Ctx, env exp.Env, e *exp.Expr) error {
	for i, arg := range e.Args {
		if i > 0 {
			b.WriteString(r.op)
		}
		err := renderBool(b, env, r.not, arg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r renderArith) Render(b bfr.Ctx, env exp.Env, e *exp.Expr) error {
	// TODO we need to insert respect operator precedence and insert parens
	for i, arg := range e.Args {
		if i > 0 {
			b.WriteString(string(r))
		}
		err := RenderEl(b, env, arg)
		if err != nil {
			return err
		}
	}
	return nil
}

func renderIf(b bfr.Ctx, env exp.Env, e *exp.Expr) error {
	b.WriteString("CASE ")
	var i int
	for i = 0; i+1 < len(e.Args); i += 2 {
		b.WriteString("WHEN ")
		err := renderBool(b, env, false, e.Args[i])
		if err != nil {
			return err
		}
		b.WriteString(" THEN ")
		err = RenderEl(b, env, e.Args[i+1])
		if err != nil {
			return err
		}
	}
	if i < len(e.Args) {
		b.WriteString(" ELSE ")
		err := RenderEl(b, env, e.Args[i])
		if err != nil {
			return err
		}
	}
	return b.Fmt(" END")
}

func (r renderEq) Render(b bfr.Ctx, env exp.Env, e *exp.Expr) error {
	// TODO mind nulls
	fst, err := renderString(env, e.Args[0])
	if err != nil {
		return err
	}
	for i, arg := range e.Args[1:] {
		if i > 0 {
			b.WriteString(" AND ")
		}
		b.WriteString(fst)
		b.WriteString(r.op)
		if !r.strict {
			err = RenderEl(b, env, arg)
			if err != nil {
				return err
			}
			continue
		}
		oth, err := renderString(env, arg)
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
	}
	return nil
}

func (r renderCmp) Render(b bfr.Ctx, env exp.Env, e *exp.Expr) error {
	// TODO mind nulls
	last, err := renderString(env, e.Args[0])
	if err != nil {
		return err
	}
	for i, arg := range e.Args[1:] {
		if i > 0 {
			b.WriteString(" AND ")
		}
		b.WriteString(last)
		b.WriteString(string(r))
		oth, err := renderString(env, arg)
		if err != nil {
			return err
		}
		b.WriteString(oth)
		last = oth
	}
	return nil
}

func renderAs(b bfr.Ctx, env exp.Env, e *exp.Expr) error {
	if len(e.Args) == 0 {
		return fmt.Errorf("empty as expression")
	}
	t, ok := e.Args[0].(typ.Type)
	if !ok {
		return fmt.Errorf("as expression must start with a type")
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
		err = RenderEl(b, env, e.Args[1])
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("not implemented")
	}
	b.WriteString("::")
	b.WriteString(ts)
	return nil
}

func renderCat(b bfr.Ctx, env exp.Env, e *exp.Expr) error {
	for i, arg := range e.Args {
		if i > 0 {
			b.WriteString(" || ")
		}
		// TODO cast to element type
		err := RenderEl(b, env, arg)
		if err != nil {
			return err
		}
	}
	return nil
}
func renderApd(b bfr.Ctx, env exp.Env, e *exp.Expr) error {
	if len(e.Args) == 0 {
		return fmt.Errorf("empty apd expression")
	}
	t := elType(e.Args[0])
	if t == typ.Void {
		return fmt.Errorf("untyped first argument in apd expression")
	}
	// either jsonb or postgres array
	ispg := t.Next().Kind&typ.MaskPrim != 0
	for i, arg := range e.Args {
		if i > 0 {
			b.WriteString(" || ")
		}
		// TODO cast to element type when ispg, otherwise jsonb
		_ = ispg
		err := RenderEl(b, env, arg)
		if err != nil {
			return err
		}
	}
	return nil
}

var layoutSet = []typ.Param{{Name: "a", Type: typ.Dict}, {Name: "unis"}}

func renderSet(b bfr.Ctx, env exp.Env, e *exp.Expr) error {
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
	err = RenderEl(b, env, e.Args[0])
	if err != nil {
		return err
	}
	if dict.Len() > 0 {
		b.WriteString(" || ")
		err = RenderLit(b, dict)
		if err != nil {
			return err
		}
	}
	for i := range rest {
		d := rest[len(rest)-i-1]
		b.WriteString(", {")
		b.WriteString(strings.ToLower(d.Name))
		b.WriteString("}, ")
		err = RenderEl(b, env, d.Args[0])
		if err != nil {
			return err
		}
		b.WriteString(", true)")
	}
	return nil
}

func renderBool(b bfr.Ctx, env exp.Env, not bool, e exp.El) error {
	var t typ.Type
	switch v := e.(type) {
	case lit.Lit:
		t = v.Typ()
	case *exp.Sym:
		t = v.Type
	case *exp.Expr:
		t = v.Rslv.Res()
	default:
		return fmt.Errorf("unexpected element %s", e)
	}
	if t.Kind == typ.KindBool && not {
		b.WriteString("NOT ")
	}
	el, err := renderString(env, e)
	if err != nil {
		return err
	}
	b.WriteString(el)
	// add boolean conversion if necessary
	if t.Kind&typ.FlagOpt != 0 {
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
	if cmp != "" {
		op := " != "
		if not {
			op = " = "
		}
		b.WriteString(op)
		b.WriteString(cmp)
		if oth != "" {
			if not {
				b.WriteString(" OR ")
			} else {
				b.WriteString(" AND ")
			}
			b.WriteString(el)
			b.WriteString(op)
			b.WriteString(oth)
		}
	}
	return nil
}

func renderString(env exp.Env, e exp.El) (string, error) {
	var b strings.Builder
	err := RenderEl(bfr.Ctx{B: &b}, env, e)
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
		return "", "", fmt.Errorf("error unexpected type %s", t)
	}
	return
}
