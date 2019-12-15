package genpg

import (
	"strings"

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
	writeFunc  func(*Writer, exp.Env, *exp.Call) error
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

func (r writeRaw) WriteCall(w *Writer, env exp.Env, e *exp.Call) error {
	restore := w.Prec(r.prec)
	w.WriteString(r.raw)
	restore()
	return nil
}
func (r writeFunc) WriteCall(w *Writer, env exp.Env, e *exp.Call) error { return r(w, env, e) }
func (r writeLogic) WriteCall(w *Writer, env exp.Env, e *exp.Call) error {
	restore := w.Prec(r.prec)
	for i, arg := range e.All() {
		if i > 0 {
			w.WriteString(r.op)
		}
		err := writeBool(w, env, r.not, arg)
		if err != nil {
			return err
		}
	}
	restore()
	return nil
}

func (r writeArith) WriteCall(w *Writer, env exp.Env, e *exp.Call) error {
	restore := w.Prec(r.prec)
	for i, arg := range e.All() {
		if i > 0 {
			w.WriteString(r.op)
		}
		err := w.WriteEl(env, arg)
		if err != nil {
			return err
		}
	}
	restore()
	return nil
}

func renderIf(w *Writer, env exp.Env, e *exp.Call) error {
	restore := w.Prec(PrecDef)
	w.WriteString("CASE ")
	var i int
	all := e.All()
	for i = 0; i+1 < len(all); i += 2 {
		w.WriteString("WHEN ")
		err := writeBool(w, env, false, all[i])
		if err != nil {
			return err
		}
		w.WriteString(" THEN ")
		err = w.WriteEl(env, all[i+1])
		if err != nil {
			return err
		}
	}
	if i < len(all) {
		w.WriteString(" ELSE ")
		err := w.WriteEl(env, all[i])
		if err != nil {
			return err
		}
	}
	w.WriteString(" END")
	restore()
	return nil
}

func (r writeEq) WriteCall(w *Writer, env exp.Env, e *exp.Call) error {
	all := e.All()
	if len(all) > 2 {
		defer w.Prec(PrecAnd)()
	}
	// TODO mind nulls
	fst, err := writeString(w, env, all[0])
	if err != nil {
		return err
	}
	for i, arg := range all[1:] {
		if i > 0 {
			w.WriteString(" AND ")
		}
		if !r.strict {
			restore := w.Prec(PrecCmp)
			w.WriteString(fst)
			w.WriteString(r.op)
			err = w.WriteEl(env, arg)
			if err != nil {
				return err
			}
			restore()
			continue
		}
		w.WriteByte('(')
		w.WriteString(fst)
		w.WriteString(r.op)
		oth, err := writeString(w, env, arg)
		if err != nil {
			return err
		}
		w.WriteString(oth)
		w.WriteString(" AND pg_typeof(")
		w.WriteString(fst)
		w.WriteByte(')')
		w.WriteString(r.op)
		w.WriteString("pg_typeof(")
		w.WriteString(oth)
		w.WriteByte(')')
		w.WriteByte(')')
	}
	return nil
}

func (r writeCmp) WriteCall(w *Writer, env exp.Env, e *exp.Call) error {
	all := e.All()
	if len(all) > 2 {
		defer w.Prec(PrecAnd)()
	}
	// TODO mind nulls
	last, err := writeString(w, env, all[0])
	if err != nil {
		return err
	}
	for i, arg := range all[1:] {
		if i > 0 {
			w.WriteString(" AND ")
		}
		restore := w.Prec(PrecCmp)
		w.WriteString(last)
		w.WriteString(string(r))
		oth, err := writeString(w, env, arg)
		if err != nil {
			return err
		}
		w.WriteString(oth)
		restore()
		last = oth
	}
	return nil
}

func writeCon(w *Writer, env exp.Env, e *exp.Call) error {
	all := e.All()
	if len(all) == 0 {
		return cor.Errorf("empty as expression")
	}
	t, ok := all[0].(*exp.Atom).Lit.(typ.Type)
	if !ok {
		return cor.Errorf("as expression must start with a type")
	}
	ts, err := TypString(t)
	if err != nil {
		return err
	}
	switch len(all) {
	case 1:
		zero, _, err := zeroStrings(t)
		if err != nil {
			return err
		}
		w.WriteString(zero)
	case 2:
		el := all[1]
		if a, ok := el.(*exp.Atom); ok {
			l, err := lit.Convert(a.Lit, t, 0)
			if err != nil {
				return err
			}
			el = &exp.Atom{Lit: l, Src: a.Src}
			return w.WriteEl(env, el)
		}
		err = w.WriteEl(env, el)
		if err != nil {
			return err
		}
	default:
		return cor.Errorf("not implemented %q", e)
	}
	w.WriteString("::")
	w.WriteString(ts)
	return nil
}

func writeCat(w *Writer, env exp.Env, e *exp.Call) error {
	restore := w.Prec(PrecDef)
	for i, arg := range e.All() {
		if i > 0 {
			w.WriteString(" || ")
		}
		// TODO cast to element type
		err := w.WriteEl(env, arg)
		if err != nil {
			return err
		}
	}
	restore()
	return nil
}
func writeApd(w *Writer, env exp.Env, e *exp.Call) error {
	all := e.All()
	if len(all) == 0 {
		return cor.Errorf("empty apd expression")
	}
	t := elType(all[0])
	if t == typ.Void {
		return cor.Errorf("untyped first argument in apd expression")
	}
	restore := w.Prec(PrecDef)
	// either jsonb or postgres array
	ispg := t.Elem().Kind&typ.KindPrim != 0
	for i, arg := range all {
		if i > 0 {
			w.WriteString(" || ")
		}
		// TODO cast to element type when ispg, otherwise jsonb
		_ = ispg
		err := w.WriteEl(env, arg)
		if err != nil {
			return err
		}
	}
	restore()
	return nil
}

var setSig = exp.MustSig("<form set @1|keyr plain?; tags?; @1>")

func writeSet(w *Writer, env exp.Env, e *exp.Call) error {
	// First arg can only be a jsonb obj
	// TODO but check that
	decls := e.Tags(2)
	// Collect literals and other decls. We can merge all literal directly,
	// but need to use jsonb_set for references and other expressions.
	dict := &lit.Dict{}
	var rest []*exp.Tag
	for _, d := range decls {
		switch v := d.El.(type) {
		case lit.Lit:
			dict.SetKey(d.Name, v)
		case *exp.Sym:
			rest = append(rest, d)
		case *exp.Call:
			rest = append(rest, d)
		}
	}
	for range rest {
		w.WriteString("jsonb_set(")
	}
	err := w.WriteEl(env, e.Arg(0))
	if err != nil {
		return err
	}
	if dict.Len() > 0 {
		w.WriteString(" || ")
		err = WriteLit(w, dict)
		if err != nil {
			return err
		}
	}
	for i := range rest {
		d := rest[len(rest)-i-1]
		w.WriteString(", {")
		w.WriteString(strings.ToLower(d.Name))
		w.WriteString("}, ")
		err = w.WriteEl(env, d.El)
		if err != nil {
			return err
		}
		w.WriteString(", true)")
	}
	return nil
}

func writeBool(w *Writer, env exp.Env, not bool, e exp.El) error {
	var t typ.Type
	switch v := e.(type) {
	case *exp.Sym:
		t = v.Type
	case *exp.Call:
		t = v.Spec.Res()
	case *exp.Atom:
		t = v.Typ()
	default:
		return cor.Errorf("unexpected element %s", e)
	}
	if t.Kind == typ.KindBool {
		if not {
			defer w.Prec(PrecNot)()
			w.WriteString("NOT ")
		}
		return w.WriteEl(env, e)
	}
	// add boolean conversion if necessary
	if t.Kind&typ.KindOpt != 0 {
		defer w.Prec(PrecIs)()
		err := w.WriteEl(env, e)
		if err != nil {
			return err
		}
		if not {
			w.WriteString(" IS NULL")
		} else {
			w.WriteString(" IS NOT NULL")
		}
		return nil
	}
	cmp, oth, err := zeroStrings(t)
	if err != nil {
		return err
	}
	if oth != "" {
		if not {
			defer w.Prec(PrecOr)()
		} else {
			defer w.Prec(PrecAnd)()
		}
	} else if cmp != "" {
		defer w.Prec(PrecCmp)()
	}
	err = w.WriteEl(env, e)
	if err != nil {
		return err
	}
	if cmp != "" {
		op := " != "
		if not {
			op = " = "
		}
		restore := w.Prec(PrecCmp)
		w.WriteString(op)
		w.WriteString(cmp)
		if oth != "" {
			if not {
				w.WriteString(" OR ")
			} else {
				w.WriteString(" AND ")
			}
			err := w.WriteEl(env, e)
			if err != nil {
				return err
			}
			w.WriteString(op)
			w.WriteString(oth)
		}
		restore()
	}
	return nil
}

func writeString(w *Writer, env exp.Env, e exp.El) (string, error) {
	cc := *w
	var b strings.Builder
	cc.Ctx = bfr.Ctx{B: &b}
	err := cc.WriteEl(env, e)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func elType(e exp.El) typ.Type {
	switch v := e.(type) {
	case *exp.Sym:
		return v.Type
	case *exp.Call:
		return v.Spec.Res()
	case lit.Lit:
		return v.Typ()
	}
	return typ.Void
}

func zeroStrings(t typ.Type) (zero, alt string, _ error) {
	switch t.Kind & typ.SlotMask {
	case typ.KindBool:
	case typ.KindNum, typ.KindInt, typ.KindReal, typ.KindBits:
		zero = "0"
	case typ.KindChar, typ.KindStr, typ.KindRaw:
		zero = "''"
	case typ.KindSpan:
		zero = "'0'"
	case typ.KindTime:
		zero = "'0001-01-01Z'"
	case typ.KindEnum:
		// TODO
	case typ.KindList:
		// TODO check if postgres array otherwise
		fallthrough
	case typ.KindIdxr:
		zero, alt = "'null'", "'[]'"
	case typ.KindKeyr, typ.KindRec, typ.KindObj:
		zero, alt = "'null'", "'{}'"
	default:
		return "", "", cor.Errorf("error unexpected type %s", t)
	}
	return
}
