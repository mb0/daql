// Package qrymem provides a query backend using in-memory go data-structures.
package qrymem

import (
	"sort"
	"strings"

	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/std"
	"github.com/mb0/xelf/typ"
)

type Backend struct {
	tables map[string]*memTable
}

func (b *Backend) Add(m *dom.Model, list *lit.List) error {
	if b.tables == nil {
		b.tables = make(map[string]*memTable)
	}
	for i, v := range list.Data {
		v, err := lit.Convert(v, m.Type, 0)
		if err != nil {
			return err
		}
		list.Data[i] = v
	}
	b.tables[m.Type.Key()] = &memTable{m.Type, list}
	return nil
}

func (b *Backend) ExecPlan(c *exp.Ctx, env exp.Env, p *qry.Plan) (*qry.Result, error) {
	res := qry.NewResult(p)
	ctx := ctx{c, &qry.ExecEnv{env, p, res}, res}
	for _, t := range p.Root {
		err := b.execTask(ctx, t, res.Data)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

type ctx struct {
	*exp.Ctx
	exp.Env
	*qry.Result
}

func (b *Backend) execTask(c ctx, t *qry.Task, par lit.Proxy) error {
	res, err := c.Prep(par, t)
	if err != nil {
		return err
	}
	if t.Query != nil {
		return b.execQuery(c, t, res)
	}
	el, err := c.Resolve(c.Env, t.Expr, t.Type)
	if err != nil {
		return err
	}
	err = res.Assign(el.(*exp.Atom).Lit)
	if err != nil {
		return err
	}
	c.SetDone(t, res)
	return nil
}

func (b *Backend) execQuery(c ctx, t *qry.Task, res lit.Proxy) (err error) {
	q := t.Query
	model, rest := modelName(q)
	m := b.tables[model]
	if m == nil {
		return cor.Errorf("mem table %s not found in %v", model, b.tables)
	}
	if q.Ref[0] == '#' {
		return m.execCount(c, t, res)
	}
	whr, null, err := prepareWhr(c.Ctx, c.Env, q)
	if err != nil {
		return err
	}
	if null { // task result must already be initialized
		c.SetDone(t, res)
		return nil
	}
	rt := t.Type
	if rt.Kind&typ.MaskElem == typ.KindList {
		rt = rt.Elem()
	}
	result := make([]lit.Lit, 0, len(m.data.Data))
	for _, l := range m.data.Data {
		if whr != nil {
			lenv := &exp.DataScope{c.Env, l}
			res, err := c.Resolve(lenv, whr, typ.Bool)
			if err != nil {
				return err
			}
			if res.(*exp.Atom).Lit != lit.True {
				continue
			}
		}
		if rest != "" {
			// handle scalar selection
			l, err = lit.Select(l, rest)
			if err != nil {
				return err
			}
			z, err := lit.Convert(l, rt, 0)
			if err != nil {
				return cor.Errorf("exec scalar query: %v", err)
			}
			result = append(result, z)
		} else {
			// TODO use proxy type if available
			z := lit.ZeroProxy(rt)
			err := b.collectSel(c, t, l, z)
			if err != nil {
				return err
			}
			result = append(result, z)
		}
	}
	if len(q.Ord) != 0 {
		err = orderResult(result, q.Ord)
		if err != nil {
			return err
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
	var l lit.Lit = lit.Null(res.Typ())
	switch q.Ref[0] {
	case '?':
		if len(result) != 0 {
			l = result[0]
		}
	case '*':
		l = &lit.List{Elem: rt, Data: result}
	}
	err = res.Assign(l)
	if err != nil {
		return cor.Errorf("qrymem: %v", err)
	}
	c.SetDone(t, res)
	return nil
}

func (m *Backend) collectSel(c ctx, tt *qry.Task, l lit.Lit, z lit.Proxy) error {
	c.Env = &qry.TaskEnv{c.Env, c.Result, tt, l}
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
			c.SetDone(t, res)
		} else {
			err := m.execTask(c, t, z)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func orderResult(list []lit.Lit, sel []qry.Ord) (res error) {
	// TODO order on more than one field
	ord := sel[0]
	sort.SliceStable(list, func(i, j int) bool {
		a, err := lit.Select(list[i], ord.Key[1:])
		if err != nil {
			if res == nil {
				res = err
			}
			return true
		}
		b, err := lit.Select(list[j], ord.Key[1:])
		if err != nil {
			if res == nil {
				res = err
			}
			return true
		}
		less, ok := lit.Less(a, b)
		if !ok {
			if res == nil {
				res = cor.Errorf("not comparable %s %s", a, b)
			}
			return true
		}
		if ord.Desc {
			return !less
		}
		return less
	})
	return res
}

type memTable struct {
	rec  typ.Type
	data *lit.List
}

func (m *memTable) execCount(c ctx, t *qry.Task, res lit.Proxy) (err error) {
	// we can ignore order and selection completely
	whr, null, err := prepareWhr(c.Ctx, c.Env, t.Query)
	if err != nil {
		return err
	}
	if null {
		return nil
	}
	var result int64
	if whr == nil {
		result = int64(len(m.data.Data))
	} else {
		for _, l := range m.data.Data {
			// skip if it does not resolve to true
			lenv := &exp.DataScope{c.Env, l}
			res, err := c.Resolve(lenv, whr, typ.Void)
			if err != nil {
				return err
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
	err = res.Assign(lit.Int(result))
	if err != nil {
		return err
	}
	c.SetDone(t, res)
	return nil
}

var boolSpeck = std.Core(":bool")

func prepareWhr(c *exp.Ctx, env exp.Env, q *qry.Query) (x exp.El, null bool, _ error) {
	if q.Whr == nil || len(q.Whr.Els) == 0 {
		return nil, false, nil
	}
	if len(q.Whr.Els) == 1 && isBool(q.Whr.Els[0]) {
		x = q.Whr.Els[0]
	}
	if x == nil {
		x = &exp.Call{Spec: boolSpeck, Args: q.Whr.Els}
	}
	res, err := c.With(true, false).Resolve(env, x, c.New())
	if err != nil {
		if err != exp.ErrUnres {
			return nil, false, err
		}
		return res.(*exp.Call), false, nil
	}
	return nil, res.(*exp.Atom).Lit != lit.True, nil
}

func modelName(q *qry.Query) (model, rest string) {
	model = q.Ref[1:]
	s := strings.SplitN(model, ".", 3)
	if len(s) < 3 {
		return model, ""
	}
	rest = s[2]
	return model[:len(model)-len(rest)-1], rest
}

func isBool(el exp.El) bool {
	t := el.Typ()
	switch t.Kind {
	case typ.KindTyp:
		t = el.(*exp.Atom).Lit.(typ.Type)
	case typ.KindSym:
		t = el.(*exp.Sym).Type
	case typ.KindCall:
		t = el.(*exp.Call).Spec.Res()
	}
	return t == typ.Bool
}
