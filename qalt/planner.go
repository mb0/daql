package qry

import (
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

// Planner is a program specific helper that manages available query backends.
type Planner interface {
	// Prep returns a query with an unresolved subject for key or nil
	Prep(key string) *Query
	// Plan evaluates the query arguments or an error.
	// If all arguments are evaluated it returns a new job call or a zero literal if the query
	// can be ommited (e.g. when whr evaluates to false).
	Plan(p *exp.Prog, env exp.Env, c *exp.Call, q *Query) (exp.El, error)
	// Exec executes a job using the backend and returns the result.
	Exec(p *exp.Prog, j *Job) (lit.Lit, error)
}

type planner struct {
	Backends
}

func (pl *planner) Prep(key string) *Query {
	q := &Query{Kind: Kind(key[0]), Planner: pl}
	q.Ref = key[1:]
	switch key[1] {
	case '.', '/', '$': // path subj
	default: // model subj
		q.Bend, q.Model = pl.Find(q.Ref)
		if q.Model == nil {
			q.Err = cor.Errorf("no subj model found for %q", key)
		}
	}
	return q
}

func (pl *planner) Plan(p *exp.Prog, env exp.Env, c *exp.Call, q *Query) (exp.El, error) {
	plan := &Plan{}
	// TODO create a job environment, so nested jobs can detect all parent jobs
	job := &Job{Plan: plan, Query: q, Env: env}
	plan.Jobs = []*Job{job}
	spec := &exp.Spec{typ.Form("job", qrySig.Params), job}
	call, err := p.NewCall(spec, nil, c.Src)
	if err != nil {
		return nil, err
	}
	return call, exp.ErrUnres
}

func (pl *planner) Exec(p *exp.Prog, j *Job) (lit.Lit, error) {
	if j.Res == nil {
		err := j.Bend.Exec(p, j.Plan)
		if err != nil {
			return nil, err
		}
	}
	return j.Res, nil
}
