package qry

import (
	"sort"

	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/std"
	"github.com/mb0/xelf/typ"
)

type LitBackend struct {
	*dom.Project
	Data map[string]*lit.List
}

func (b *LitBackend) Proj() *dom.Project {
	return b.Project
}

func (b *LitBackend) Exec(p *exp.Prog, pl *Plan) error {
	for _, j := range pl.Jobs {
		err := b.execJob(p, j)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *LitBackend) execJob(p *exp.Prog, j *Job) error {
	key := j.Model.Qualified()
	list := b.Data[key]
	if list == nil {
		return cor.Errorf("lit backend query data %q not found", key)
	}
	res, err := execListQry(p, j, list)
	if err != nil {
		return err
	}
	j.Res = res
	return nil
}

func (b *LitBackend) Add(m *dom.Model, list *lit.List) error {
	if b.Data == nil {
		b.Data = make(map[string]*lit.List)
	}
	for i, v := range list.Data {
		v, err := lit.Convert(v, m.Type, 0)
		if err != nil {
			return err
		}
		list.Data[i] = v
	}
	b.Data[m.Qualified()] = list
	return nil
}

func execLocalQuery(p *exp.Prog, j *Job) (lit.Lit, error) {
	sym := &exp.Sym{Name: j.Ref}
	_, err := p.Eval(j.Env, sym, typ.Void)
	if err != nil {
		return nil, err
	}
	list := getList(sym.Lit)
	if list == nil {
		return nil, cor.Errorf("literal query expects list got %T", sym.Lit)
	}
	return execListQry(p, j, list)
}

func getList(l lit.Lit) *lit.List {
	switch v := l.(type) {
	case *lit.List:
		return v
	case lit.Indexer:
		ls := make([]lit.Lit, 0, v.Len())
		v.IterIdx(func(idx int, l lit.Lit) error {
			ls = append(ls, l)
			return nil
		})
		return &lit.List{Data: ls}
	}
	return nil
}

var andSpec = std.Core("and")

func execListQry(p *exp.Prog, j *Job, list *lit.List) (lit.Lit, error) {
	var whr exp.El
	if len(j.Whr) > 0 {
		whr = &exp.Dyn{Els: append([]exp.El{&exp.Atom{Lit: andSpec}}, j.Whr...)}
	}
	if j.Kind == KindCount {
		return collectCount(p, j, list, whr)
	}
	res, err := collectList(p, j, list, whr)
	if err != nil {
		return nil, err
	}
	switch j.Kind {
	case KindOne:
		if len(res.Data) == 0 {
			return lit.Nil, nil
		}
		return res.Data[0], nil
	case KindMany:
		return res, nil
	}
	return nil, cor.Errorf("exec unknown query kind %s", j.Kind)
}

func collectList(p *exp.Prog, j *Job, list *lit.List, whr exp.El) (*lit.List, error) {
	res := make([]lit.Lit, 0, len(list.Data))
	org := list.Data
	if whr != nil {
		org = make([]lit.Lit, 0, len(list.Data))
	}
	for _, l := range list.Data {
		if whr != nil {
			ok, err := filter(p, j.Env, l, whr)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			org = append(org, l)
		}
		if len(j.Sel.Fields) == 0 {
			res = append(res, l)
			continue
		}
		rec := l.(lit.Keyer)
		px := lit.ZeroProxy(j.Sel.Type)
		z, ok := px.(lit.Keyer)
		for _, f := range j.Sel.Fields {
			var val lit.Lit
			var err error
			if f.El != nil {
				env := &exp.DataScope{j.Env, exp.Def{l.Typ(), l}}
				el, err := p.Eval(env, f.El, f.Type)
				if len(f.Nest) > 0 && err == exp.ErrUnres {
					el, err = p.Eval(env, el, f.Type)
				}
				if err != nil {
					return nil, err
				}
				val = el.(*exp.Atom).Lit
			} else {
				val, err = rec.Key(f.Key)
				if err != nil {
					return nil, err
				}
			}
			if ok {
				_, err = z.SetKey(f.Key, val)
			} else {
				err = px.Assign(val)
			}
			if err != nil {
				return nil, err
			}
		}
		res = append(res, px)
	}
	if len(j.Ord) != 0 {
		err := orderResult(res, org, j.Ord)
		if err != nil {
			return nil, err
		}
	}
	if j.Off > 0 {
		if len(res) > int(j.Off) {
			res = res[j.Off:]
		} else {
			res = nil
		}
	}
	if j.Lim > 0 && len(res) > int(j.Lim) {
		res = res[:j.Lim]
	}
	return &lit.List{Data: res}, nil
}

func collectCount(p *exp.Prog, j *Job, list *lit.List, whr exp.El) (lit.Lit, error) {
	// we can ignore order and selection completely
	var res int64
	if whr == nil {
		res = int64(len(list.Data))
	} else {
		for _, l := range list.Data {
			ok, err := filter(p, j.Env, l, whr)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			res++
		}
	}
	if j.Off > 0 {
		if res > j.Off {
			res -= j.Off
		} else {
			res = 0
		}
	}
	if j.Lim > 0 && res > j.Lim {
		res = j.Lim
	}
	return lit.Int(res), nil
}

func filter(p *exp.Prog, env exp.Env, l lit.Lit, whr exp.El) (bool, error) {
	env = &exp.DataScope{env, exp.Def{l.Typ(), l}}
	res, err := p.Eval(env, whr, typ.Void)
	if err != nil {
		return false, err
	}
	return res.(*exp.Atom).Lit == lit.True, nil
}

func orderResult(sel, subj []lit.Lit, ords []Ord) (res error) {
	sort.Stable(orderer{sel, subj, func(i, j int) bool {
		less, err := orderFunc(sel, subj, i, j, ords)
		if err != nil && res == nil {
			res = err
		}
		return less
	}})
	return res
}

func orderFunc(sel, subj []lit.Lit, i, j int, ords []Ord) (bool, error) {
	ord := ords[0]
	list := sel
	if ord.Subj {
		list = subj
	}
	a, err := lit.Select(list[i], ord.Key)
	if err != nil {
		return true, err
	}
	b, err := lit.Select(list[j], ord.Key)
	if err != nil {
		return true, err
	}
	less, same, ok := lit.Comp(a, b)
	if !ok {
		return true, cor.Errorf("not orderable literals %s %s", a, b)
	}
	if same && len(ords) > 1 {
		return orderFunc(sel, subj, i, j, ords[1:])
	}
	if ord.Desc {
		return !less, nil
	}
	return less, nil
}

type orderer struct {
	a, b []lit.Lit
	less func(i, j int) bool
}

func (o orderer) Len() int { return len(o.a) }
func (o orderer) Swap(i, j int) {
	o.a[i], o.a[j] = o.a[j], o.a[i]
	o.b[i], o.b[j] = o.b[j], o.b[i]
}
func (o orderer) Less(i, j int) bool {
	return o.less(i, j)
}
