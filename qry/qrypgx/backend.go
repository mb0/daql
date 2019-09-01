// Package qrypgx provides a query backend using postgresql database using the pgx client package.
package qrypgx

import (
	"io"
	"strings"

	"github.com/jackc/pgx"
	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/mig"
	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

// Backend is a specialized postresql backend using the pgx package.
type Backend struct {
	DB *pgx.ConnPool
	mig.Record
	tables map[string]*dom.Model
}

func New(db *pgx.ConnPool, proj *dom.Project) *Backend {
	tables := make(map[string]*dom.Model, len(proj.Schemas)*8)
	for _, s := range proj.Schemas {
		for _, m := range s.Models {
			if m.Type.Kind == typ.KindObj {
				continue
			}
			// TODO check if model is actually part of the database
			tables[m.Qualified()] = m
		}
	}
	return &Backend{DB: db, Record: mig.Record{Project: proj}, tables: tables}
}

func (b *Backend) Exec(c *exp.Prog, env exp.Env, doc *qry.Doc) (lit.Lit, error) {
	p, err := Analyse(doc)
	if err != nil {
		return nil, err
	}
	denv := doc.EvalEnv(env)
	ctx := &execer{b, c, denv, denv, nil}
	for _, j := range p.Jobs {
		err := ctx.execJob(j, denv.Data)
		if err != nil {
			return nil, err
		}
	}
	return denv.Data, nil
}

var _ mig.Dataset = (*Backend)(nil)

// Close satisfies the dataset interface but does not close the underlying connection pool.
func (b *Backend) Close() error { return nil }

func (b *Backend) Keys() []string {
	res := make([]string, 0, len(b.tables))
	for k := range b.tables {
		res = append(res, k)
	}
	return res
}

func (b *Backend) Iter(key string) (mig.Iter, error) {
	m := b.tables[key]
	if m != nil {
		return openRowsIter(b.DB, m)
	}
	return nil, cor.Errorf("no table with key %s", key)
}

func openRowsIter(db *pgx.ConnPool, m *dom.Model) (*rowsIter, error) {
	res, err := lit.MakeRec(m.Type)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("SELECT ")
	for i, kv := range res.List {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(kv.Key)
	}
	b.WriteString(" FROM ")
	b.WriteString(m.Qualified())

	rows, err := db.Query(b.String())
	if err != nil {
		return nil, err
	}
	return &rowsIter{rows, res, nil}, err
}

type rowsIter struct {
	*pgx.Rows
	res  *lit.Rec
	args []interface{}
}

func (it *rowsIter) Scan() (lit.Lit, error) {
	if !it.Next() {
		return nil, io.EOF
	}
	res := it.res.New().(*lit.Rec)
	if it.args == nil {
		it.args = make([]interface{}, len(res.List))
	}
	args := it.args[:]
	for _, kv := range res.List {
		args = append(args, kv.Lit.(lit.Proxy).Ptr())
	}
	err := it.Rows.Scan(args...)
	if err != nil {
		return nil, err
	}
	return res, nil
}
func (it *rowsIter) Close() error { it.Rows.Close(); return nil }
