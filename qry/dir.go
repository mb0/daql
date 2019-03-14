package qry

import (
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
)

type SimpleDirector struct {
	Backend
}

func (d *SimpleDirector) ExecPlan(c *exp.Ctx, env exp.Env, p *Plan) error {
	if len(p.Root) == 0 {
		return cor.Error("empty plan")
	}
	if p.Simple {
		t := p.Root[0]
		t.Result = p.Result
		return d.execTask(c, env, t)
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
		return d.execTask(c, env, t)
	}
	return nil
}

func (d *SimpleDirector) execTask(c *exp.Ctx, env exp.Env, t *Task) error {
	if t.Query != nil {
		err := d.Backend.ExecQuery(c, env, t)
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
