// Package pg provides conversions to postgres literals, expressions.
package pg

import (
	"fmt"
	"strings"

	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
)

// RenderEl renders the element e to b or returns an error.
// This is used for explicit selectors for example.
func RenderEl(b bfr.Ctx, env exp.Env, e exp.El) error {
	switch v := e.(type) {
	case lit.Lit:
		return RenderLit(b, v)
	case *exp.Ref:
		// is this a column name?
		return RenderRef(b, env, v)
	case *exp.Expr:
		return RenderExpr(b, env, v)
	}
	return fmt.Errorf("unexpected element %[1]T %[1]s", e)
}

// RenderRef renders the reference r to b using env or returns an error.
// References can point schema objects like tables, enums and columns, or point inside composite
// typed columns like arrays, jsonb or previous results. It uses env to determine how to render
// the reference.
func RenderRef(b bfr.Ctx, env exp.Env, r *exp.Ref) error {
	// TODO we need check the name, but for now work with the key as is
	b.WriteString(r.Name)
	return nil
}

// RenderExpr renders the expression e to b using env or returns an error.
// Most xelf expressions with resolvers from the core or lib built-ins have a corresponding
// expression in postgresql. Custom resolvers can be rendered to sql by detecting
// and handling them before calling this function.
func RenderExpr(b bfr.Ctx, env exp.Env, e *exp.Expr) error {
	r := rendrMap[e.Name]
	if r != nil {
		return r.Render(b, env, e)
	}
	// dyn and reduce are not supported
	// TODO let and with might use common table expressions on a higher level
	return fmt.Errorf("not implemented for %s", e.Name)
}

type exprRenderer interface {
	Render(bfr.Ctx, exp.Env, *exp.Expr) error
}

var rendrMap map[string]exprRenderer

func init() {
	rendrMap = map[string]exprRenderer{
		// I found no better way sql expression to fail when resolved but not otherwise.
		// Sadly we cannot transport any failure message, but it suffices, because this is
		// only meant to be a test helper.
		"fail":  renderConst("1.2=3/0"), // 3..2..1..boom!
		"if":    renderFunc(renderIf),
		"and":   renderLogic{" AND ", false},
		"or":    renderLogic{" OR ", false},
		"bool":  renderLogic{" AND ", false},
		"not":   renderLogic{" AND ", true},
		"add":   renderArith(" + "),
		"sub":   renderArith(" - "),
		"mul":   renderArith(" * "),
		"div":   renderArith(" / "),
		"eq":    renderEq{" = ", false},
		"ne":    renderEq{" != ", false},
		"equal": renderEq{" = ", true},
		"lt":    renderCmp(" < "),
		"gt":    renderCmp(" > "),
		"le":    renderCmp(" <= "),
		"ge":    renderCmp(" >= "),
		"as":    renderFunc(renderAs),
		"cat":   renderFunc(renderCat),
		"apd":   renderFunc(renderApd),
		"set":   renderFunc(renderSet),
	}
}

func writeSuffix(b bfr.Ctx, l lit.Lit, fix string) error {
	err := l.WriteBfr(b)
	if err != nil {
		return err
	}
	return b.Fmt(fix)
}

func writeJSONB(b bfr.Ctx, l lit.Lit) error {
	var bb strings.Builder
	err := l.WriteBfr(bfr.Ctx{B: &bb, JSON: true})
	if err != nil {
		return err
	}
	writeQuote(b, bb.String())
	return b.Fmt("::jsonb")
}

func writeArray(b bfr.Ctx, l lit.Arr) error {
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
	t, err := TypString(l.Elem())
	if err != nil {
		return err
	}
	b.WriteString("::")
	b.WriteString(t)
	b.WriteString("[]")
	return nil
}

// writeQuote quotes a string as a postgres string, all single quotes are use sql escaping.
func writeQuote(b bfr.Ctx, text string) {
	b.WriteByte('\'')
	b.WriteString(strings.Replace(text, "'", "''", -1))
	b.WriteByte('\'')
}
