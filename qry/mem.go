package qry

import (
	"log"
	"sort"
	"strings"

	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

type MemBackend struct {
	tables map[string]*memTable
}

func (b *MemBackend) Add(m *dom.Model, arg interface{}) error {
	if b.tables == nil {
		b.tables = make(map[string]*memTable)
	}
	ref := strings.ToLower(m.Ref())
	a, err := lit.Proxy(arg)
	if err != nil {
		return err
	}
	l, err := lit.Convert(a, typ.List, 0)
	if err != nil {
		return err
	}
	b.tables[ref] = &memTable{m.Typ(), l.(lit.List)}
	return nil
}

func (b *MemBackend) ExecQuery(c *exp.Ctx, env exp.Env, t *Task) (err error) {
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
	if rt.Kind&typ.MaskElem == typ.KindArr {
		rt = rt.Next()
	} else {
		rt, _ = rt.Deopt()
	}
	result := make(lit.List, 0, len(m.data))
	for _, l := range m.data {
		if whr != nil {
			lenv := &LitEnv{env, l}
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
		}
		// TODO use proxy type if available
		z := lit.ZeroProxy(rt)
		err = b.collectSel(c, env, t, z, l)
		if err != nil {
			return err
		}
		result = append(result, z)
	}
	if len(q.Ord) != 0 {
		// TODO sort result by ord keys
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

func (m *MemBackend) collectSel(c *exp.Ctx, env exp.Env, tt *Task, a lit.Assignable, l lit.Lit) (err error) {
	sel := tt.Query.Sel
	if len(sel) == 0 { // return subject
		return lit.AssignTo(l, a)
	}
	keyer, ok := a.(lit.Keyer)
	if !ok {
		return cor.Errorf("expect keyer got %s", a.Typ())
	}
	tenv := &TaskEnv{env, tt, l}
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
			err = m.ExecQuery(c, tenv, t)
		} else {
			res, err = lit.Select(l, key)
		}
		if err != nil {
			return err
		}
		err = keyer.SetKey(key, res.(lit.Lit))
		if err != nil {
			return err
		}
	}
	return nil
}
func orderResult(list lit.List, sel []Ord) (res error) {
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
		log.Printf("less %v %s < %s == %v %v", ord, a, b, less, ok)
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
	data lit.List
}

func (m *memTable) execCount(c *exp.Ctx, env exp.Env, t *Task) (err error) {
	// we can ignore order and selection completely
	whr, null, err := prepareWhr(env, t.Query)
	if err != nil {
		return err
	}
	if null {
		return nil
	}
	if whr == nil {
		return t.Result.Assign(lit.Int(len(m.data)))
	}
	var result int
	for _, l := range m.data {
		// skip if it does not resolve to true
		lenv := &LitEnv{env, l}
		res, err := andForm.Resolve(c, lenv, whr, typ.Bool)
		if err != nil {
			return err
		}
		if res != lit.True {
			continue
		}
		result++
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

var andForm *exp.Form

func init() {
	andForm = exp.Core("and").(*exp.Form)
}

func prepareWhr(env exp.Env, q *Query) (x *exp.Expr, null bool, _ error) {
	if len(q.Whr) == 0 {
		return nil, false, nil
	}
	x = &exp.Expr{andForm, q.Whr, typ.Bool}
	// TODO use an appropriate environment
	res, err := exp.Resolve(env, x)
	if err != nil {
		if err != exp.ErrUnres {
			return nil, false, err
		}
		return res.(*exp.Expr), false, nil
	}
	return nil, res != lit.True, nil
}

func modelName(q *Query) (model, rest string) {
	model = q.Ref[1:]
	s := strings.SplitN(model, ".", 3)
	if len(s) < 3 {
		return model, ""
	}
	rest = s[2]
	return model[:len(model)-len(rest)-1], rest
}
