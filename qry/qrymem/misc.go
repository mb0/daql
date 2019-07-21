package qrymem

import (
	"sort"
	"strings"

	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/std"
	"github.com/mb0/xelf/typ"
)

var boolSpeck = std.Core(":bool")

func prepareWhr(q *qry.Query) (x exp.El, null bool, _ error) {
	if q.Whr == nil || len(q.Whr.Els) == 0 {
		return nil, false, nil
	}
	if len(q.Whr.Els) == 1 && isBool(q.Whr.Els[0]) {
		x = q.Whr.Els[0]
	}
	if x == nil {
		x = &exp.Call{Spec: boolSpeck, Args: q.Whr.Els}
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
