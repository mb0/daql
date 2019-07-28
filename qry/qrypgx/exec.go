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
	*qry.DocEnv
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
	qs, ps, err := genQueryStr(e.Ctx, e.Env, j)
	if err != nil {
		return err
	}
	var args []interface{}
	if len(ps) != 0 {
		args = make([]interface{}, 0, len(ps))
		for _, p := range ps {
			if p.Value != nil {
				args = append(args, p.Value)
				continue
			}
			return cor.Errorf("unexpected external param %+v", p)
		}
	}
	rows, err := e.DB.Query(qs, args...)
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
	e.Done(j.Task, res)
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
	Alias map[*qry.Task]string
	*qry.Task
	Env    exp.Env
	Prefix bool
}

func (je *jobEnv) Parent() exp.Env      { return je.Env }
func (je *jobEnv) Supports(x byte) bool { return x == '.' }
func (je *jobEnv) Get(sym string) *exp.Def {
	if sym[0] != '.' {
		return nil
	}
	dots := 1
	n := sym[1:]
	if n[0] == '.' {
		return nil
	}
	sp := strings.SplitN(n, ".", 2)
	if dots == 1 {
		// TODO only check if inline or joined query
		for _, s := range je.Query.Sel {
			if s.Name != sp[0] {
				continue
			}
			return &exp.Def{Type: s.Type}
		}
	} else {
		return nil
	}
	p, _, err := je.Query.Type.ParamByKey(sp[0])
	if err != nil {
		return nil
	}
	return &exp.Def{Type: p.Type}
}

func (je *jobEnv) prefix(n string, t *qry.Task) string {
	if !je.Prefix {
		return n
	}
	return fmt.Sprintf("%s.%s", je.Alias[t], n)
}

type jobTranslator struct{}

func (jt jobTranslator) Translate(env exp.Env, s *exp.Sym) (string, lit.Lit, error) {
	switch s.Name[0] {
	case '/', '$':
		d := exp.LookupSupports(env, s.Name, s.Name[0])
		if d == nil {
			return "", nil, exp.ErrUnres
		}
		return s.Name, d.Lit, genpg.External
	case '.':
	default:
		return genpg.ExpEnv{}.Translate(env, s)
	}
	env = exp.Supports(env, '.')
	if env == nil {
		return s.Name, nil, cor.Errorf("no dot scope for %s", s.Name)
	}
	je, ok := env.(*jobEnv)
	if !ok {
		return genpg.ExpEnv{}.Translate(env, s)
	}
	dots := 1
	n := s.Name[1:]
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
