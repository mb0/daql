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

func (b *Backend) Add(m *dom.Model, list lit.Idxr) error {
	if b.tables == nil {
		b.tables = make(map[string]*memTable)
	}
	for i, v := range list {
		v, err := lit.Convert(v, m.Type, 0)
		if err != nil {
			return err
		}
		list[i] = v
	}
	b.tables[m.Type.Key()] = &memTable{m.Type, list}
	return nil
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

func (b *Backend) execQuery(c *exp.Ctx, env exp.Env, t *qry.Task) (err error) {
	q := t.Query
	model, rest := modelName(q)
	m := b.tables[model]
	if m == nil {
		return cor.Errorf("mem table %s not found in %v", model, b.tables)
	}
	if q.Ref[0] == '#' {
		return m.execCount(c, env, t)
	}
	whr, null, err := prepareWhr(env, q)
	if err != nil {
		return err
	}
	if null { // task result must already be initialized
		return nil
	}
	rt := t.Type
	if rt.Kind&typ.MaskElem == typ.KindList {
		rt = rt.Elem()
	} else {
		rt, _ = rt.Deopt()
	}
	result := make(lit.Idxr, 0, len(m.data))
	for _, l := range m.data {
		if whr != nil {
			lenv := &exp.DataScope{env, l}
			res, err := andForm.Resolve(c, lenv, whr, typ.Bool)
			if err != nil {
				return err
			}
			if res != lit.True {
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
				return err
			}
			result = append(result, z)
		} else {
			// TODO use proxy type if available
			z := lit.ZeroProxy(rt)
			err = b.collectSel(c, env, t, z, l)
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
		if len(result) > q.Off {
			result = result[q.Off:]
		} else {
			result = nil
		}
	}
	if q.Lim > 0 && len(result) > q.Lim {
		result = result[:q.Lim]
	}
	switch q.Ref[0] {
	case '?':
		if len(result) != 0 {
			err := lit.AssignTo(result[0], t.Result)
			return err
		}
		return nil
	}
	return t.Result.Assign(result)
}

func (m *Backend) collectSel(c *exp.Ctx, env exp.Env, tt *qry.Task, a lit.Assignable, l lit.Lit,
) (err error) {
	keyer, ok := a.(lit.Keyer)
	if !ok {
		return cor.Errorf("expect keyer got %s", a.Typ())
	}
	tenv := &qry.TaskEnv{env, tt, l}
	sel := tt.Query.Sel
	for _, t := range sel {
		key := strings.ToLower(t.Name)
		var res exp.El
		if t.Expr != nil {
			res, err = c.Resolve(tenv, t.Expr, t.Type)
		} else if t.Query != nil {
			res, err = keyer.Key(key)
			if err != nil {
				return err
			}
			t.Result, ok = res.(lit.Assignable)
			if !ok {
				return cor.Errorf("expect assignable got %T", res)
			}
			err = m.execQuery(c, tenv, t)
		} else {
			res, err = lit.Select(l, key)
		}
		if err != nil {
			return err
		}
		_, err = keyer.SetKey(key, res.(lit.Lit))
		if err != nil {
			return err
		}
	}
	return nil
}
func orderResult(list lit.Idxr, sel []qry.Ord) (res error) {
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
	data lit.Idxr
}

func (m *memTable) execCount(c *exp.Ctx, env exp.Env, t *qry.Task) (err error) {
	// we can ignore order and selection completely
	whr, null, err := prepareWhr(env, t.Query)
	if err != nil {
		return err
	}
	if null {
		return nil
	}
	var result int
	if whr == nil {
		result = len(m.data)
	} else {
		for _, l := range m.data {
			// skip if it does not resolve to true
			lenv := &exp.DataScope{env, l}
			res, err := andForm.Resolve(c, lenv, whr, typ.Bool)
			if err != nil {
				return err
			}
			if res != lit.True {
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
	return t.Result.Assign(lit.Int(result))
}

var andForm *exp.Spec

func init() {
	andForm = std.Core("and")
}

func prepareWhr(env exp.Env, q *qry.Query) (x *exp.Call, null bool, _ error) {
	if len(q.Whr.Els) == 0 {
		return nil, false, nil
	}
	x = &exp.Call{Spec: andForm, Args: q.Whr.Els}
	res, err := exp.Resolve(env, x)
	if err != nil {
		if err != exp.ErrUnres {
			return nil, false, err
		}
		return res.(*exp.Call), false, nil
	}
	return nil, res != lit.True, nil
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
