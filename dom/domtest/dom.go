// Package domtest has default schemas and helpers for testing.
package domtest

import (
	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/mig"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lit"
)

type Fixture struct {
	dom.Project
	mig.Manifest
	Fix *lit.Dict
}

func New(raw, fix string) (*Fixture, error) {
	res := &Fixture{}
	env := dom.NewEnv(dom.Env, &res.Project)
	_, err := dom.ExecuteString(env, raw)
	if err != nil {
		return nil, cor.Errorf("schema: %w", err)
	}
	res.Manifest, err = res.Manifest.Update(&res.Project)
	if err != nil {
		return nil, cor.Errorf("manifest: %w", err)
	}
	l, err := lit.ParseString(fix)
	if err != nil {
		return nil, cor.Errorf("fixture: %w", err)
	}
	res.Fix = l.(*lit.Dict)
	return res, nil
}

func Must(pro *Fixture, err error) *Fixture {
	if err != nil {
		panic(err)
	}
	return pro
}

func (f *Fixture) Version() mig.Version { return f.First() }
func (f *Fixture) Keys() []string       { return f.Fix.Keys() }
func (f *Fixture) Close() error         { return nil }
func (f *Fixture) Iter(key string) (mig.Iter, error) {
	l, _ := f.Fix.Key(key)
	idxr, ok := l.(lit.Indexer)
	if !ok {
		return nil, cor.Errorf("want idxr got %T", l)
	}
	return &idxrIter{idxr, 0}, nil
}

type idxrIter struct {
	lit.Indexer
	idx int
}

func (it *idxrIter) Close() error { return nil }

func (it *idxrIter) Scan() (lit.Lit, error) {
	l, err := it.Idx(it.idx)
	it.idx++
	return l, err
}
