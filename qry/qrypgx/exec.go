package qrypgx

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx"
	"github.com/mb0/daql/gen/genpg"
	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
)

type execer struct {
	*Backend
	*exp.Ctx
	exp.Env
	*qry.Result
	args []interface{}
}

func (e *execer) execJob(j *Job, par lit.Proxy) error {
	res, err := e.Prep(par, j.Task)
	if err != nil {
		return err
	}
	if j.Query == nil {
		el, err := e.Resolve(e.Env, j.Expr, j.Type)
		if err != nil {
			return err
		}
		return res.Assign(el.(lit.Lit))
	}
	qs, err := genQueryStr(e.Ctx, e.Env, j)
	if err != nil {
		return err
	}
	// TODO collect and pass query parameters
	rows, err := e.DB.Query(qs)
	if err != nil {
		return cor.Errorf("query %s: %w", qs, err)
	}
	defer rows.Close()
	switch {
	case j.Kind&KindScalar != 0:
		err = e.scanScalar(j, res, rows)
	case j.Kind&KindSingle != 0:
		err = e.scanOne(j, res, rows)
	case j.Kind&KindMulti != 0:
		err = e.scanMany(j, res, rows)
	default:
		return cor.Errorf("unexpected query kind for %s", j.Query.Ref)
	}
	if err != nil {
		return err
	}
	e.SetDone(j.Task, res)
	return nil
}

func (e *execer) scanScalar(j *Job, res lit.Proxy, rows *pgx.Rows) error {
	if !rows.Next() {
		return cor.Errorf("no result for query %s", j.Query.Ref)
	}
	err := rows.Scan(res.Ptr())
	if err != nil {
		return cor.Errorf("scan row for query %s: %w", j.Query.Ref, err)
	}
	if rows.Next() {
		return cor.Errorf("additional results for query %s", j.Query.Ref)
	}
	return rows.Err()
}

func (e *execer) scanOne(j *Job, res lit.Proxy, rows *pgx.Rows) error {
	if rows.Next() {
		err := e.scanRow(j, res, rows)
		if err != nil {
			return err
		}
	}
	if rows.Next() {
		return cor.Errorf("additional results for query %s", j.Query.Ref)
	}
	return rows.Err()
}

func (e *execer) scanMany(j *Job, res lit.Proxy, rows *pgx.Rows) error {
	a, ok := lit.Deopt(res).(lit.Appender)
	if !ok {
		return cor.Errorf("expect arr result got %T", res)
	}
	for rows.Next() {
		el, err := a.Element()
		if err != nil {
			return err
		}
		err = e.scanRow(j, el, rows)
		if err != nil {
			return err
		}
		a, err = a.Append(el)
		if err != nil {
			return err
		}
	}
	return rows.Err()
}

func (e *execer) scanRow(j *Job, r lit.Lit, rows *pgx.Rows) (err error) {
	k, ok := lit.Deopt(r).(lit.Keyer)
	if !ok {
		return cor.Errorf("expect keyer result got %T", r)
	}
	if cap(e.args) < len(j.Cols) {
		e.args = make([]interface{}, 0, len(j.Cols))
	} else {
		e.args = e.args[:0]
	}
	for _, c := range j.Cols {
		var el lit.Lit
		if c.Job == j || c.Job.Kind&KindInlined != 0 {
			el, err = k.Key(c.Key)
			if err != nil {
				return cor.Errorf("prep result field %s: %w", c.Key, err)
			}
		} else { // joined table
			el, err = k.Key(strings.ToLower(c.Job.Name))
			if err != nil {
				return cor.Errorf("prep result field %s: %w", c.Key, err)
			}
			if c.Job.Kind&KindScalar == 0 {
				sub, ok := lit.Deopt(el).(lit.Keyer)
				if !ok {
					return cor.Errorf("expect keyer result got %T", r)
				}
				el, err = sub.Key(c.Key)
				if err != nil {
					return cor.Errorf("prep result field %s: %w", c.Key, err)
				}
			}
		}
		p, ok := el.(lit.Proxy)
		if !ok {
			return cor.Errorf("expect assignable result got %T", el)
		}
		e.args = append(e.args, p.Ptr())
	}
	err = rows.Scan(e.args...)
	if err != nil {
		return cor.Errorf("scan row for query %s: %w", j.Query.Ref, err)
	}
	return nil
}

type jobEnv struct {
	*Job
	*qry.Task
	exp.Env
	Prefix bool
}

func (je *jobEnv) Translate(s *exp.Sym) (string, lit.Lit, error) {
	n := s.Name
	switch n[0] {
	case '/', '$':
		return n, nil, genpg.External
	case '.':
		// handled after switch
	default:
		d := je.Get(n)
		if d == nil {
			return n, nil, exp.ErrUnres
		}
		if d.Lit != nil {
			return "", d.Lit, nil
		}
		return n, nil, genpg.External
	}
	dots := 1
	n = n[1:]
	t := je.Task
	for n != "" && n[0] == '.' {
		if t.Parent == nil {
			return "", nil, cor.Errorf("no parent for relative path %s in %v",
				n, t)
		}
		n = n[1:]
		t = t.Parent
	}
	sp := strings.SplitN(n, ".", 2)
	if dots == 1 {
		// TODO only check if inline or joined query
		for _, s := range t.Query.Sel {
			if s.Name != sp[0] {
				continue
			}
			return n, nil, nil
			// TODO format correct name
		}
	} else {
		return "", nil, cor.Errorf("no selection for %q", s.Name)
	}
	_, _, err := t.Query.Type.ParamByKey(sp[0])
	if err != nil {
		return "", nil, cor.Errorf("no selection or field for %q in %s: %w", s.Name, t.Query.Type, err)
	}
	return je.prefix(n, t), nil, nil
}

func (je *jobEnv) prefix(n string, t *qry.Task) string {
	if !je.Prefix || je.Job == nil {
		return n
	}
	return fmt.Sprintf("%s.%s", je.Alias[t], n)
}
