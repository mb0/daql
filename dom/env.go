package dom

import (
	"strings"

	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/utl"
)

var Env = exp.Builtin{
	utl.StrLib.Lookup(),
	utl.TimeLib.Lookup(),
	exp.Std, exp.Core,
}

type ProjectEnv struct {
	pa exp.Env
	pr *Project
	sm map[string]*SchemaEnv
}

func NewProjectEnv(env exp.Env) *ProjectEnv {
	return &ProjectEnv{pa: env, pr: &Project{},
		sm: make(map[string]*SchemaEnv)}
}

func (s *ProjectEnv) Parent() exp.Env                      { return s.pa }
func (s *ProjectEnv) Def(sym string, r exp.Resolver) error { return exp.ErrNoDefEnv }
func (s *ProjectEnv) Get(sym string) exp.Resolver {
	if sym == "schema" {
		return &SchemaEnv{Project: s}
	}
	split := strings.SplitN(sym, ".", 2)
	se := s.sm[split[0]]
	if se == nil { // we found no schema, query parent
		return s.pa.Get(sym)

	}
	if len(split) == 1 { // schema ref
		return se
	}
	return se.Get(split[1])
}

type SchemaEnv struct {
	Project *ProjectEnv
	Node    utl.Node
	Schema  *Schema
	mm      map[string]*ModelEnv
}

func (s *SchemaEnv) Parent() exp.Env                      { return s.Project }
func (s *SchemaEnv) Def(sym string, r exp.Resolver) error { return exp.ErrNoDefEnv }

func (r *SchemaEnv) Get(sym string) exp.Resolver {
	split := strings.SplitN(sym, ".", 2)
	me := r.mm[split[0]]
	if me == nil { // we found no model, query parent
		return r.Project.Get(sym)
	}
	if len(split) == 1 { // model ref
		return utl.LitResolver{me.Model.Typ()}
	}
	return me.Get(split[1])
}

func (r *SchemaEnv) Resolve(c *exp.Ctx, env exp.Env, e exp.El) (exp.El, error) {
	if r.Node != nil {
		return r.Node, nil
	}
	x, ok := e.(*exp.Expr)
	if !ok {
		return e, exp.ErrUnres
	}
	err := resolveSchema(c, r, "", x.Args)
	if err != nil {
		return e, err
	}
	r.Project.sm[r.Schema.Name] = r
	return r.Node, nil
}

type ModelEnv struct {
	*SchemaEnv
	Node  utl.Node
	Model *Model
}

func (r *ModelEnv) Get(sym string) exp.Resolver {
	// TODO check local names
	return r.SchemaEnv.Get(sym)
}
