package qry

import (
	"strings"

	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

type Plan struct {
	Jobs []*Job
}

// Job describes a concrete query execution with all arguments evaluated.
type Job struct {
	*Plan
	*Query
	Env exp.Env
	Res lit.Lit
}

func (j *Job) Resl(p *exp.Prog, env exp.Env, c *exp.Call, h typ.Type) (exp.El, error) {
	// jobs are already resolved
	return c, nil
}
func (j *Job) Eval(p *exp.Prog, env exp.Env, c *exp.Call, h typ.Type) (exp.El, error) {
	res, err := j.Planner.Exec(p, j)
	if err != nil {
		return c, err
	}
	return &exp.Atom{Lit: res, Src: c.Src}, nil
}

type Backend interface {
	Proj() *dom.Project
	Exec(*exp.Prog, *Plan) error
}

type Backends []Backend

func (bs Backends) Find(key string) (Backend, *dom.Model) {
	for _, b := range bs {
		if m := findModel(b.Proj(), key); m != nil {
			return b, m
		}
	}
	return nil, nil
}

func findModel(p *dom.Project, key string) *dom.Model {
	if p == nil {
		return nil
	}
	sp := strings.SplitN(key, ".", 2)
	if len(sp) == 2 {
		return p.Schema(sp[0]).Model(sp[1])
	}
	for _, s := range p.Schemas {
		if m := s.Model(key); m != nil {
			return m
		}
	}
	return nil
}
