package qrypgx

import (
	"log"
	"strings"

	"github.com/jackc/pgx"
	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/gen"
	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
)

// Backend is a specialized postresql backend using the pgx package.
type Backend struct {
	DB   *pgx.ConnPool
	Proj *dom.Project
}

func New(db *pgx.ConnPool, proj *dom.Project) *Backend {
	return &Backend{DB: db, Proj: proj}
}

func (b *Backend) ExecPlan(c *exp.Ctx, env exp.Env, p *qry.Plan) error {
	if p.Simple {
		t := p.Root[0]
		t.Result = p.Result
		return b.execTask(c, env, t)
	}
	keyer, ok := p.Result.(lit.Keyer)
	if !ok {
		return cor.Errorf("want keyer plan result got %T", p.Result)
	}
	for _, t := range p.Root {
		r, err := keyer.Key(strings.ToLower(t.Name))
		if err != nil {
			return err
		}
		t.Result, ok = r.(lit.Assignable)
		if !ok {
			return cor.Errorf("want assignable task result got %s from %T", r, keyer)
		}
		err = b.execTask(c, env, t)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Backend) execTask(c *exp.Ctx, env exp.Env, t *qry.Task) error {
	if t.Query != nil {
		err := b.execQuery(c, env, t)
		if err != nil {
			return err
		}
	} else {
		el, err := c.Resolve(env, t.Expr, t.Type)
		if err != nil {
			return err
		}
		err = t.Result.Assign(el.(lit.Lit))
		if err != nil {
			return err
		}
	}
	t.Done = true
	return nil
}

func (b *Backend) execQuery(c *exp.Ctx, env exp.Env, t *qry.Task) error {
	q := t.Query
	schema, model, rest := splitName(q)
	m := b.Proj.Schema(schema).Model(model)
	if m == nil {
		return cor.Errorf("schema model for %s not found", q.Ref)
	}
	if rest != "" {
		return cor.Errorf("field query not yet implemented for %s", q.Ref)
	}
	var sb strings.Builder
	// write query.
	// XXX could we cache and clearly identify prepared statements?
	err := genQuery(&gen.Ctx{Ctx: bfr.Ctx{B: &sb}}, c, env, t)
	if err != nil {
		return err
	}
	qs := sb.String()
	rows, err := b.DB.Query(qs)
	if err != nil {
		return cor.Errorf("query %s: %w", qs, err)
	}
	switch q.Ref[0] {
	case '#':
		if !rows.Next() {
			return cor.Errorf("no result for count query %s", q.Ref)
		}
		err = rows.Scan(t.Result.Ptr())
		if err != nil {
			return cor.Errorf("scan: %w", err)
		}
		if rows.Next() {
			return cor.Errorf("additional results for count query")
		}
		return rows.Err()
	case '?':
		if !rows.Next() {
			return rows.Err()
		}
		k, ok := lit.Deopt(t.Result).(lit.Keyer)
		if !ok {
			return cor.Errorf("expect keyer result got %T", t.Result)
		}
		args := make([]interface{}, 0, len(q.Sel))
		for _, s := range q.Sel {
			el, err := k.Key(strings.ToLower(s.Name))
			if err != nil {
				return cor.Errorf("prep scan: %w", err)
			}
			v, ok := el.(lit.Assignable)
			if !ok {
				return cor.Errorf("expect assignable result got %T", el)
			}
			args = append(args, v.Ptr())
		}
		log.Printf("scan query %q with args %v", qs, args)
		err = rows.Scan(args...)
		if err != nil {
			return cor.Errorf("scan: %w", err)
		}
		if rows.Next() {
			return cor.Errorf("additional results for count query")
		}
		return rows.Err()
	}
	// result should be an assignable arr
	a, ok := lit.Deopt(t.Result).(lit.Appender)
	if !ok {
		return cor.Errorf("expect arr result got %T", t.Result)
	}
	n := a.Len()
	args := make([]interface{}, len(q.Sel))
	for rows.Next() {
		null := lit.ZeroProxy(a.Typ().Elem())
		a, err = a.Append(null)
		if err != nil {
			return err
		}
		el, err := a.Idx(n)
		if err != nil {
			return err
		}
		k, ok := lit.Deopt(el).(lit.Keyer)
		if !ok {
			return cor.Errorf("expect keyer result got %T", el)
		}
		args = args[:0]
		for _, s := range q.Sel {
			el, err := k.Key(strings.ToLower(s.Name))
			if err != nil {
				return cor.Errorf("prep scan: %w", err)
			}
			v, ok := el.(lit.Assignable)
			if !ok {
				return cor.Errorf("expect assignable result got %T", el)
			}
			args = append(args, v.Ptr())
		}
		log.Printf("scan query %q with args %v", qs, args)
		err = rows.Scan(args...)
		if err != nil {
			return cor.Errorf("scan: %w", err)
		}
		n++
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return t.Result.Assign(a)
}

func splitName(q *qry.Query) (schema, model, rest string) {
	s := strings.SplitN(q.Ref[1:], ".", 3)
	if len(s) < 2 {
		return q.Ref[1:], "", ""
	}
	if len(s) > 2 {
		rest = s[2]
	}
	return s[0], s[1], rest
}
