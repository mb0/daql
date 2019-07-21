// Package qrymem provides a query backend using in-memory go data-structures.
package qrymem

import (
	"io"

	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/mig"
	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
)

type Backend struct {
	mig.Record
	tables map[string]*lit.List
}

var _ mig.Dataset = (*Backend)(nil)

func (b *Backend) Close() error { return nil }

func (b *Backend) Keys() []string {
	res := make([]string, 0, len(b.tables))
	for key := range b.tables {
		res = append(res, key)
	}
	return res
}

func (b *Backend) Iter(key string) (mig.Iter, error) {
	list := b.tables[key]
	if list != nil {
		return &listIter{List: list}, nil
	}
	return nil, cor.Errorf("no table with key %s", key)
}

func (b *Backend) Add(m *dom.Model, list *lit.List) error {
	if b.tables == nil {
		b.tables = make(map[string]*lit.List)
	}
	for i, v := range list.Data {
		v, err := lit.Convert(v, m.Type, 0)
		if err != nil {
			return err
		}
		list.Data[i] = v
	}
	b.tables[m.Type.Key()] = list
	return nil
}

func (b *Backend) Exec(c *exp.Ctx, env exp.Env, p *qry.Doc) (*qry.Result, error) {
	res := qry.NewResult(p)
	e := execer{b, c, &qry.ExecEnv{env, p, res}, res}
	for _, t := range p.Root {
		err := execTask(e, t, res.Data)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

type listIter struct {
	*lit.List
	Idx int
}

func (it *listIter) Scan() (lit.Lit, error) {
	if i := it.Idx; i < len(it.Data) {
		it.Idx++
		return it.Data[i], nil
	}
	return nil, io.EOF
}

func (it *listIter) Close() error {
	it.Idx = len(it.Data)
	return nil
}
