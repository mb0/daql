package dom

import (
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
	"github.com/mb0/xelf/utl"
)

var Env = exp.Builtin{
	utl.StrLib.Lookup(),
	utl.TimeLib.Lookup(),
	exp.Std, exp.Core,
}

// ProjectEnv is a root environment that allows schema declaration or model resolution.
type ProjectEnv struct {
	pa exp.Env
	pr *Project
}

func NewEnv(parent exp.Env, project *Project) *ProjectEnv {
	return &ProjectEnv{pa: parent, pr: project}
}

func (s *ProjectEnv) Parent() exp.Env                      { return s.pa }
func (s *ProjectEnv) Def(sym string, r exp.Resolver) error { return exp.ErrNoDefEnv }
func (s *ProjectEnv) Get(sym string) exp.Resolver {
	if sym == "schema" {
		return &SchemaEnv{Project: s}
	}
	prefix := sym[0] == '~'
	if prefix { // strip prefix and continue
		sym = sym[1:]
	}
	split := strings.SplitN(sym, ".", 3)
	ss := s.pr.Schema(split[0])
	if ss == nil && !prefix { // no schema found query parent if not prefixed
		return s.pa.Get(sym)
	}
	if len(split) == 1 || s == nil {
		return nil
	}
	m := ss.Model(split[1])
	if len(split) == 2 || m == nil {
		return utl.LitResolver{m.Typ()}
	}
	if m.Kind&typ.MaskPrim != 0 {
		c := m.Const(split[2])
		if c != nil {
			return utl.LitResolver{constLit(m, c)}
		}
	} else {
		f := m.Field(split[2])
		if f != nil {
			return utl.TypedUnresolver{f.Type}
		}
	}
	return nil
}

func constLit(m *Model, c *cor.Const) lit.Lit {
	if m.Kind != typ.KindEnum {
		return lit.FlagInt{m.Typ(), lit.Int(c.Val)}
	}
	return lit.EnumStr{m.Typ(), lit.Str(strings.ToLower(c.Name))}
}

// SchemaEnv is used inside schema definitions.
// It resolves previously declared models.
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
	m := r.Schema.Model(split[0])
	if m == nil {
		r.Project.Get(sym)
	}
	if len(split) == 1 {
		return utl.LitResolver{m.Typ()}
	}
	return (&ModelEnv{r, m}).Get(split[1])
}

func (r *SchemaEnv) Resolve(c *exp.Ctx, env exp.Env, e exp.El) (exp.El, error) {
	x, ok := e.(*exp.Expr)
	if !ok {
		return e, exp.ErrUnres
	}
	err := resolveSchema(c, r, "", x.Args)
	if err != nil {
		return e, err
	}
	return r.Node, nil
}

// ModelEnv is used inside model definitions.
// It resolves previously declarted fields or constants.
type ModelEnv struct {
	*SchemaEnv
	Model *Model
}

func (r *ModelEnv) Get(sym string) exp.Resolver {
	if r.Model.Kind&typ.MaskPrim != 0 {
		c := r.Model.Const(sym)
		if c != nil {
			return utl.LitResolver{constLit(r.Model, c)}
		}
	} else {
		f := r.Model.Field(sym)
		if f != nil {
			return utl.TypedUnresolver{f.Type}
		}
	}
	return r.SchemaEnv.Get(sym)
}
