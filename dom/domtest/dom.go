// Package domtest has default schemas and helpers for testing.
package domtest

import (
	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lit"
)

type Proj struct {
	dom.Project
	Fix *lit.Dict
}

func Fixture(raw, fix string) (*Proj, error) {
	res := &Proj{}
	env := dom.NewEnv(dom.Env, &res.Project)
	_, err := dom.ExecuteString(env, raw)
	if err != nil {
		return nil, cor.Errorf("schema: %w", err)
	}
	l, err := lit.ParseString(fix)
	if err != nil {
		return nil, cor.Errorf("fixture: %w", err)
	}
	res.Fix = l.(*lit.Dict)
	return res, nil
}
