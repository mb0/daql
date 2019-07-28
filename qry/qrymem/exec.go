package qrymem

import (
	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

type execer struct {
	*Backend
	*exp.Ctx
	exp.Env
	*qry.DocEnv
}

func execTask(c execer, t *qry.Task, par lit.Proxy) error {
	res, err := c.Prep(par, t)
	if err != nil {
		return err
	}
	if t.Query == nil {
		return execExpr(c, t, res)
	}
	return execQuery(c, t, res)
}

func execExpr(c execer, t *qry.Task, res lit.Proxy) error {
	el, err := c.Resolve(c.Env, t.Expr, t.Type)
	if err != nil {
		return err
	}
	err = res.Assign(el.(*exp.Atom).Lit)
	if err != nil {
		return err
	}
	c.Done(t, res)
	return nil
}

func execQuery(c execer, t *qry.Task, res lit.Proxy) error {
	model, rest := modelName(t.Query)
	m := c.tables[model]
	if m == nil {
		return cor.Errorf("mem table %s not found in %v", model, c.tables)
	}
	whr, null, err := prepareWhr(t.Query)
	if err != nil {
		return err
	}
	if !null { // else task result is already initialized
		var l lit.Lit
		var list *lit.List
		switch t.Query.Ref[0] {
		case '?':
			list, err = collectList(c, t, m, whr, rest)
			if err != nil {
				return cor.Errorf("qrymem: collect list %v", err)
			}
			if list.Len() != 0 {
				l = list.Data[0]
			} else {
				l = lit.Null(res.Typ())
			}
		case '*':
			l, err = collectList(c, t, m, whr, rest)
		case '#':
			l, err = collectCount(c, t, m, whr)
		}
		if err != nil {
			return cor.Errorf("qrymem: %v", err)
		}
		err = res.Assign(l)
		if err != nil {
			return cor.Errorf("qrymem assign: %v", err)
		}
	}
	c.Done(t, res)
	return nil
}

func collectSel(c execer, tt *qry.Task, l lit.Lit, z lit.Proxy) error {
	c.Env = &qry.TaskEnv{c.Env, c.DocEnv, tt, l}
	for _, t := range tt.Query.Sel {
		if t.Query == nil && t.Expr == nil {
			el, err := lit.Select(l, cor.Keyed(t.Name))
			if err != nil {
				return err
			}
			res, err := c.Prep(z, t)
			if err != nil {
				return err
			}
			err = res.Assign(el.(lit.Lit))
			if err != nil {
				return err
			}
			c.Done(t, res)
		} else {
			err := execTask(c, t, z)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func collectList(c execer, t *qry.Task, m *lit.List, whr exp.El, rest string) (*lit.List, error) {
	q := t.Query
	rt := t.Type
	if rt.Kind&typ.MaskElem == typ.KindList {
		rt = rt.Elem()
	}
	result := make([]lit.Lit, 0, len(m.Data))
	for _, l := range m.Data {
		if whr != nil {
			lenv := &exp.DataScope{c.Env, l}
			res, err := c.Resolve(lenv, whr, typ.Bool)
			if err != nil {
				return nil, err
			}
			if res.(*exp.Atom).Lit != lit.True {
				continue
			}
		}
		if rest != "" {
			// handle scalar selection
			var err error
			l, err = lit.Select(l, rest)
			if err != nil {
				return nil, err
			}
			z, err := lit.Convert(l, rt, 0)
			if err != nil {
				return nil, cor.Errorf("exec scalar query: %v", err)
			}
			result = append(result, z)
		} else {
			// TODO use proxy type if available
			z := lit.ZeroProxy(rt)
			err := collectSel(c, t, l, z)
			if err != nil {
				return nil, err
			}
			result = append(result, z)
		}
	}
	if len(q.Ord) != 0 {
		err := orderResult(result, q.Ord)
		if err != nil {
			return nil, err
		}
	}
	if q.Off > 0 {
		if len(result) > int(q.Off) {
			result = result[q.Off:]
		} else {
			result = nil
		}
	}
	if q.Lim > 0 && len(result) > int(q.Lim) {
		result = result[:q.Lim]
	}
	return &lit.List{Elem: rt, Data: result}, nil
}

func collectCount(c execer, t *qry.Task, m *lit.List, whr exp.El) (lit.Lit, error) {
	// we can ignore order and selection completely
	var result int64
	if whr == nil {
		result = int64(len(m.Data))
	} else {
		for _, l := range m.Data {
			// skip if it does not resolve to true
			lenv := &exp.DataScope{c.Env, l}
			res, err := c.Resolve(lenv, whr, typ.Void)
			if err != nil {
				return nil, err
			}
			if res.(*exp.Atom).Lit != lit.True {
				continue
			}
			result++
		}
	}
	q := t.Query
	if q.Off > 0 {
		if result > q.Off {
			result -= q.Off
		} else {
			result = 0
		}
	}
	if q.Lim > 0 && result > q.Lim {
		result = q.Lim
	}
	return lit.Int(result), nil
}
