// Package genpg provides code generation helpers postgresql query and schema generation.
package genpg

import (
	"strings"

	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
)

var External = cor.StrError("external symbol")

// WriteEl writes the element e to b or returns an error.
// This is used for explicit selectors for example.
func (w *Writer) WriteEl(env exp.Env, e exp.El) error {
	switch v := e.(type) {
	case *exp.Sym:
		n, l, err := w.Translate(env, v)
		if err != nil {
			return cor.Errorf("symbol %q: %w", v.Name, err)
		}
		if l != nil {
			return WriteLit(w, v.Lit)
		}
		return writeIdent(w, n)
	case *exp.Call:
		return w.WriteExpr(env, v)
	case *exp.Atom:
		return WriteLit(w, v.Lit)
	}
	return cor.Errorf("unexpected element %[1]T %[1]s", e)
}

// WriteExpr writes the expression e to b using env or returns an error.
// Most xelf expressions with resolvers from the core or lib built-ins have a corresponding
// expression in postgresql. Custom resolvers can be rendered to sql by detecting
// and handling them before calling this function.
func (w *Writer) WriteExpr(env exp.Env, e *exp.Call) error {
	key := e.Spec.Key()
	if key == "bool" {
		key = ":bool"
	}
	r := exprWriterMap[key]
	if r != nil {
		return r.WriteCall(w, env, e)
	}
	// dyn and reduce are not supported
	// TODO let and with might use common table expressions on a higher level
	return cor.Errorf("no writer for expression %s", e)
}

type callWriter interface {
	WriteCall(*Writer, exp.Env, *exp.Call) error
}

var exprWriterMap map[string]callWriter

func init() {
	exprWriterMap = map[string]callWriter{
		// I found no better way sql expression to fail when resolved but not otherwise.
		// Sadly we cannot transport any failure message, but it suffices, because this is
		// only meant to be a test helper.
		"fail":  writeRaw{"3.2=1/0", PrecCmp}, // 3..2..1..boom!
		"if":    writeFunc(renderIf),
		"and":   writeLogic{" AND ", false, PrecAnd},
		"or":    writeLogic{" OR ", false, PrecOr},
		":bool": writeLogic{" AND ", false, PrecAnd},
		"not":   writeLogic{" AND ", true, PrecAnd},
		"add":   writeArith{" + ", PrecAdd},
		"sub":   writeArith{" - ", PrecAdd},
		"mul":   writeArith{" * ", PrecMul},
		"div":   writeArith{" / ", PrecMul},
		"eq":    writeEq{" = ", false},
		"ne":    writeEq{" != ", false},
		"equal": writeEq{" = ", true},
		"lt":    writeCmp(" < "),
		"gt":    writeCmp(" > "),
		"le":    writeCmp(" <= "),
		"ge":    writeCmp(" >= "),
		"as":    writeFunc(writeAs),
		"cat":   writeFunc(writeCat),
		"apd":   writeFunc(writeApd),
		"set":   writeFunc(writeSet),
	}
}

const (
	_ = iota
	PrecOr
	PrecAnd
	PrecNot
	PrecIs  // , is null, is not null, …
	PrecCmp // <, >, =, <=, >=, <>, !=
	PrecIn  // , between, like, ilike, similar
	PrecDef
	PrecAdd // +, -
	PrecMul // *, /, %
)

func writeSuffix(w *Writer, l lit.Lit, fix string) error {
	err := l.WriteBfr(&w.Ctx)
	if err != nil {
		return err
	}
	return w.Fmt(fix)
}

func writeJSONB(w *Writer, l lit.Lit) error {
	var bb strings.Builder
	err := l.WriteBfr(&bfr.Ctx{B: &bb, JSON: true})
	if err != nil {
		return err
	}
	WriteQuote(w, bb.String())
	return w.Fmt("::jsonb")
}

func writeArray(w *Writer, l lit.Appender) error {
	var bb strings.Builder
	bb.WriteByte('{')
	err := l.IterIdx(func(i int, e lit.Lit) error {
		if i > 0 {
			bb.WriteByte(',')
		}
		return e.WriteBfr(&bfr.Ctx{B: &bb, JSON: true})
	})
	if err != nil {
		return err
	}
	bb.WriteByte('}')
	WriteQuote(w, bb.String())
	t, err := TypString(l.Typ().Elem())
	if err != nil {
		return err
	}
	w.WriteString("::")
	w.WriteString(t)
	w.WriteString("[]")
	return nil
}

// WriteQuote quotes a string as a postgres string, all single quotes are use sql escaping.
func WriteQuote(w bfr.B, text string) {
	w.WriteByte('\'')
	w.WriteString(strings.Replace(text, "'", "''", -1))
	w.WriteByte('\'')
}

var keywords map[string]struct{}

func writeIdent(w *Writer, name string) error {
	name = strings.ToLower(name)
	if _, ok := keywords[name]; !ok {
		return w.Fmt(name)
	}
	w.WriteByte('"')
	w.WriteString(name)
	return w.WriteByte('"')
}
