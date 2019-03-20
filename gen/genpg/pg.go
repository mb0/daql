// Package genpg provides conversions to postgres literals, expressions.
package genpg

import (
	"strings"

	"github.com/mb0/daql/gen"
	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
)

// WriteEl writes the element e to b or returns an error.
// This is used for explicit selectors for example.
func WriteEl(b *gen.Ctx, env exp.Env, e exp.El) error {
	switch v := e.(type) {
	case lit.Lit:
		return WriteLit(b, v)
	case *exp.Sym:
		// is this a column name?
		return WriteRef(b, env, v)
	case *exp.Expr:
		return WriteExpr(b, env, v)
	}
	return cor.Errorf("unexpected element %[1]T %[1]s", e)
}

// WriteRef writes the reference r to b using env or returns an error.
// References can point schema objects like tables, enums and columns, or point inside composite
// typed columns like arrays, jsonb or previous results. It uses env to determine how to render
// the reference.
func WriteRef(b *gen.Ctx, env exp.Env, r *exp.Sym) error {
	// TODO we need check the name, but for now work with the key as is
	name := r.Name
	if name[0] == '.' {
		name = name[1:]
	}
	b.WriteString(name)
	return nil
}

// WriteExpr writes the expression e to b using env or returns an error.
// Most xelf expressions with resolvers from the core or lib built-ins have a corresponding
// expression in postgresql. Custom resolvers can be rendered to sql by detecting
// and handling them before calling this function.
func WriteExpr(b *gen.Ctx, env exp.Env, e *exp.Expr) error {
	key := e.Rslv.Key()
	r := exprWriterMap[key]
	if r != nil {
		return r.WriteExpr(b, env, e)
	}
	// dyn and reduce are not supported
	// TODO let and with might use common table expressions on a higher level
	return cor.Errorf("not implemented for %s", key)
}

type exprWriter interface {
	WriteExpr(*gen.Ctx, exp.Env, *exp.Expr) error
}

var exprWriterMap map[string]exprWriter

func init() {
	exprWriterMap = map[string]exprWriter{
		// I found no better way sql expression to fail when resolved but not otherwise.
		// Sadly we cannot transport any failure message, but it suffices, because this is
		// only meant to be a test helper.
		"fail":  writeRaw{"3.2=1/0", PrecCmp}, // 3..2..1..boom!
		"if":    writeFunc(renderIf),
		"and":   writeLogic{" AND ", false, PrecAnd},
		"or":    writeLogic{" OR ", false, PrecOr},
		"bool":  writeLogic{" AND ", false, PrecAnd},
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

func writeSuffix(b *gen.Ctx, l lit.Lit, fix string) error {
	err := l.WriteBfr(b.Ctx)
	if err != nil {
		return err
	}
	return b.Fmt(fix)
}

func writeJSONB(b *gen.Ctx, l lit.Lit) error {
	var bb strings.Builder
	err := l.WriteBfr(bfr.Ctx{B: &bb, JSON: true})
	if err != nil {
		return err
	}
	writeQuote(b, bb.String())
	return b.Fmt("::jsonb")
}

func writeArray(b *gen.Ctx, l lit.Arr) error {
	var bb strings.Builder
	bb.WriteByte('{')
	err := l.IterIdx(func(i int, e lit.Lit) error {
		if i > 0 {
			bb.WriteByte(',')
		}
		return e.WriteBfr(bfr.Ctx{B: &bb, JSON: true})
	})
	if err != nil {
		return err
	}
	bb.WriteByte('}')
	writeQuote(b, bb.String())
	t, err := TypString(l.Element())
	if err != nil {
		return err
	}
	b.WriteString("::")
	b.WriteString(t)
	b.WriteString("[]")
	return nil
}

// writeQuote quotes a string as a postgres string, all single quotes are use sql escaping.
func writeQuote(b bfr.B, text string) {
	b.WriteByte('\'')
	b.WriteString(strings.Replace(text, "'", "''", -1))
	b.WriteByte('\'')
}

var keywords map[string]struct{}

func writeIdent(b *gen.Ctx, name string) error {
	name = strings.ToLower(name)
	if _, ok := keywords[name]; !ok {
		return b.Fmt(name)
	}
	b.WriteByte('"')
	b.WriteString(name)
	return b.WriteByte('"')
}
