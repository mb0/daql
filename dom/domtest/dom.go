// Package domtest has default schemas and helpers for testing.
package domtest

import (
	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lit"
)

const ProdRaw = `(schema 'prod'
	(+Cat   +ID   int :pk
		+Name str)
	(+Prod  +ID   int :pk
		+Name str
		+Cat  int)
	(+Label	+ID   int :pk
		+Name str
		+Tmpl raw)
)`

type Cat struct {
	ID   int
	Name string
}

type Prod struct {
	ID   int
	Name string
	Cat  int
}

type Label struct {
	ID   int
	Name string
	Tmpl []byte
}

const ProdFixRaw = `{
	cat:[
		[25 'y']
		[2  'b']
		[3  'c']
		[1  'a']
		[4  'd']
		[26 'z']
		[24 'x']
	]
	prod:[
		[25 'Y' 1]
		[2  'B' 2]
		[3  'C' 3]
		[1  'A' 3]
		[4  'D' 2]
		[26 'Z' 1]
	]
	label:[
		[1 'M' 'foo']
		[2 'N' 'bar']
		[3 'O' 'spam']
		[4 'P' 'egg']
	]
}`

type ProdProj struct {
	dom.Project
	ProdFix *lit.Keyr
}

func ProdFixture() (*ProdProj, error) {
	res := &ProdProj{}
	env := dom.NewEnv(dom.Env, &res.Project)
	_, err := dom.ExecuteString(env, ProdRaw)
	if err != nil {
		return nil, cor.Errorf("schema: %w", err)
	}
	l, err := lit.ParseString(ProdFixRaw)
	if err != nil {
		return nil, cor.Errorf("fixture: %w", err)
	}
	res.ProdFix = l.(*lit.Keyr)
	return res, nil
}
