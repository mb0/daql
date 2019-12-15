package qrymem

import (
	"sort"

	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/std"
	"github.com/mb0/xelf/typ"
)

var andSpeck = std.Core("and")

func prepareWhr(q *qry.Query) (x exp.El, null bool, _ error) {
	if q.Whr == nil || len(q.Whr.Els) == 0 {
		return nil, false, nil
	}
	if len(q.Whr.Els) == 1 && isBool(q.Whr.Els[0]) {
		x = q.Whr.Els[0]
	} else {
		x = &exp.Dyn{
			Els: append([]exp.El{&exp.Atom{Lit: andSpeck}}, q.Whr.Els...),
			Src: q.Whr.Src,
		}
	}
	return x, false, nil
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
