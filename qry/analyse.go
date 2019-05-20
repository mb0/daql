package qry

import "github.com/mb0/xelf/exp"

// analyseDeps populates the dependencies for each task in the query plan or returns an error.
func analyseDeps(pl *Plan) (err error) {
	a := analyser{Plan: pl}
	for _, t := range pl.Root {
		if t.Query != nil {
			err = a.query(t)
			// the query expression need to be checked like expression tasks
		} else {
			// expression can either contain references to previous query results
			// or nested queries
			err = a.expr(t, nil, t.Expr)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

type analyser struct {
	exp.Ghost
	*Plan
	tmp []Dep
}

func (a *analyser) VisitSym(s *exp.Sym) error {
	if s.Name[0] == '/' {
		t, _, err := RootTask(a.Plan, s.Name)
		if err != nil {
			return err
		}
		a.tmp = append(a.tmp, Dep{Task: t})
	}
	return nil
}

func (a *analyser) query(t *Task) (err error) {
	for _, s := range t.Query.Sel {
		if s.Query != nil {
			// sub-queries always depend on the parent query
			// be it to merge in results
			s.Deps = append(s.Deps, Dep{Task: t})
		} else {
			err = a.expr(s, t, s.Expr)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *analyser) expr(t, p *Task, x exp.El) error {
	if x == nil {
		return nil
	}
	// collect root tasks referenced by an absolute path.
	// TODO handle relative paths to other queries
	a.tmp = nil
	x.Traverse(a)
	t.Deps = a.tmp
	return nil
}
