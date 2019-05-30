package qrypgx

import (
	"log"
	"strings"

	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lit"
)

func execTask(c ctx, t *qry.Task, par lit.Proxy) error {
	res, err := c.Prep(par, t)
	if err != nil {
		return err
	}
	if t.Query != nil {
		return execQuery(c, t, res)
	}
	el, err := c.Resolve(c.Env, t.Expr, t.Type)
	if err != nil {
		return err
	}
	return res.Assign(el.(lit.Lit))
}

func execQuery(c ctx, t *qry.Task, res lit.Proxy) error {
	q := t.Query
	schema, model, rest := splitName(q)
	m := c.Project.Schema(schema).Model(model)
	if m == nil {
		return cor.Errorf("schema model for %s not found", q.Ref)
	}
	if rest != "" {
		return cor.Errorf("field query not yet implemented for %s", q.Ref)
	}
	// write query.
	// XXX could we cache and clearly identify prepared statements?
	qs, err := genQueryStr(c.Ctx, c.Env, t.Query)
	if err != nil {
		return err
	}
	rows, err := c.DB.Query(qs)
	if err != nil {
		return cor.Errorf("query %s: %w", qs, err)
	}
	defer rows.Close()
	switch q.Ref[0] {
	case '#':
		if !rows.Next() {
			return cor.Errorf("no result for count query %s", q.Ref)
		}
		err = rows.Scan(res.Ptr())
		if err != nil {
			return cor.Errorf("scan: %w", err)
		}
		if rows.Next() {
			return cor.Errorf("additional results for count query")
		}
		if err = rows.Err(); err != nil {
			return err
		}
		c.SetDone(t, res)
		return nil
	case '?':
		if !rows.Next() {
			return rows.Err()
		}
		k, ok := lit.Deopt(res).(lit.Keyer)
		if !ok {
			return cor.Errorf("expect keyer result got %T", res)
		}
		args := make([]interface{}, 0, len(q.Sel))
		for _, s := range q.Sel {
			el, err := k.Key(strings.ToLower(s.Name))
			if err != nil {
				return cor.Errorf("prep scan: %w", err)
			}
			v, ok := el.(lit.Proxy)
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
		if err = rows.Err(); err != nil {
			return err
		}
		c.SetDone(t, res)
		return nil
	}
	// result should be an assignable arr
	a, ok := lit.Deopt(res).(lit.Appender)
	if !ok {
		return cor.Errorf("expect arr result got %T", res)
	}
	nn := a.Len()
	args := make([]interface{}, len(q.Sel))
	for rows.Next() {
		null := lit.ZeroProxy(a.Typ().Elem())
		a, err = a.Append(null)
		if err != nil {
			return err
		}
		el, err := a.Idx(nn)
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
			v, ok := el.(lit.Proxy)
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
		nn++
	}
	if err = rows.Err(); err != nil {
		return err
	}
	c.SetDone(t, res)
	return nil
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
