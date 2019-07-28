package qrypgx

import (
	"strings"

	"github.com/jackc/pgx"
	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/gen/genpg"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

type DB interface {
	Begin() (*pgx.Tx, error)
}

type C interface {
	Query(string, ...interface{}) (*pgx.Rows, error)
	QueryRow(string, ...interface{}) *pgx.Row
	Exec(string, ...interface{}) (pgx.CommandTag, error)
	Prepare(string, string) (*pgx.PreparedStatement, error)
	CopyFrom(pgx.Identifier, []string, pgx.CopyFromSource) (int, error)
}

func Open(dsn string, logger pgx.Logger) (*pgx.ConnPool, error) {
	conf, err := pgx.ParseDSN(dsn)
	if err != nil {
		return nil, cor.Errorf("parsing postgres dsn: %w", err)
	}
	if logger != nil {
		conf.Logger = logger
		conf.LogLevel = pgx.LogLevelWarn
	}
	db, err := pgx.NewConnPool(pgx.ConnPoolConfig{ConnConfig: conf})
	if err != nil {

		return nil, cor.Errorf("creating pgx connection pool: %w", err)
	}
	_, err = db.Exec("SELECT 1")
	if err != nil {
		db.Close()
		return nil, cor.Errorf("opening first pgx connection: %w", err)
	}
	return db, nil
}

func WithTx(db DB, f func(C) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = f(tx)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func CreateProject(db *pgx.ConnPool, p *dom.Project) error {
	return WithTx(db, func(tx C) error {
		err := dropProject(tx, p)
		if err != nil {
			return err
		}
		for _, s := range p.Schemas {
			_, err = tx.Exec("CREATE SCHEMA " + s.Name)
			if err != nil {
				return err
			}
			for _, m := range s.Models {
				err = CreateModel(tx, s, m)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func DropProject(db *pgx.ConnPool, p *dom.Project) error {
	return WithTx(db, func(tx C) error {
		return dropProject(tx, p)
	})
}

func CreateModel(tx C, s *dom.Schema, m *dom.Model) error {
	switch m.Type.Kind {
	case typ.KindBits:
		return nil
	case typ.KindEnum:
		return createModel(tx, m, (*genpg.Writer).WriteEnum)
	case typ.KindObj:
		err := createModel(tx, m, (*genpg.Writer).WriteTable)
		if err != nil {
			return err
		}
		// TODO indices
		return nil
	}
	return cor.Errorf("unexpected model kind %s", m.Type.Kind)
}

func createModel(tx C, m *dom.Model, f func(*genpg.Writer, *dom.Model) error) error {
	var b strings.Builder
	w := genpg.NewWriter(&b, genpg.ExpEnv{})
	err := f(w, m)
	if err != nil {
		return err
	}
	_, err = tx.Exec(b.String())
	return err
}

func dropProject(tx C, p *dom.Project) error {
	for i := len(p.Schemas) - 1; i >= 0; i-- {
		s := p.Schemas[i]
		_, err := tx.Exec("DROP SCHEMA IF EXISTS " + s.Name + " CASCADE")
		if err != nil {
			return err
		}
	}
	return nil
}

func CopyFrom(db *pgx.ConnPool, s *dom.Schema, fix *lit.Dict) error {
	return WithTx(db, func(tx C) error {
		for _, kl := range fix.List {
			m := s.Model(kl.Key)
			cols := modelColumns(m)
			src := &litCopySrc{List: kl.Lit.(*lit.List), typ: m.Type, cols: cols}
			_, err := tx.CopyFrom(pgx.Identifier{m.Qual(), m.Key()}, cols, src)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

type litCopySrc struct {
	*lit.List
	typ  typ.Type
	cols []string
	nxt  int
	err  error
	res  interface{}
}

func (c *litCopySrc) Next() bool {
	c.nxt++
	return c.err == nil && c.nxt <= len(c.Data)
}
func (c *litCopySrc) Values() ([]interface{}, error) {
	el, err := c.Idx(c.nxt - 1)
	if err != nil {
		c.err = err
		return nil, err
	}
	el, err = lit.Convert(el, c.typ, 0)
	if err != nil {
		c.err = err
		return nil, err
	}
	k, ok := el.(lit.Keyer)
	if !ok {
		c.err = cor.Errorf("expect keyer got %T", el)
		return nil, c.err
	}
	res := make([]interface{}, 0, len(c.cols))
	for _, col := range c.cols {
		el, err = k.Key(col)
		if err != nil {
			c.err = err
			return nil, err
		}
		v, ok := el.(interface{ Val() interface{} })
		if !ok {
			c.err = cor.Errorf("expect valuer got %T", el)
			return nil, c.err
		}
		res = append(res, v.Val())
	}
	return res, nil
}

func (c *litCopySrc) Err() error {
	return c.err
}

func modelColumns(m *dom.Model) []string {
	res := make([]string, 0, len(m.Type.Params))
	for _, p := range m.Type.Params {
		res = append(res, p.Key())
	}
	return res
}
