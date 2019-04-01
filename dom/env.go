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
	*Project
}

func NewEnv(parent exp.Env, project *Project) *ProjectEnv {
	return &ProjectEnv{pa: parent, Project: project}
}

func FindEnv(env exp.Env) *ProjectEnv {
	for env != nil {
		env = exp.Supports(env, '~')
		if p, ok := env.(*ProjectEnv); ok {
			return p
		}
		if env != nil {
			env = env.Parent()
		}
	}
	return nil
}

func (s *ProjectEnv) Parent() exp.Env      { return s.pa }
func (s *ProjectEnv) Supports(x byte) bool { return x == '~' }

func (s *ProjectEnv) Def(sym string, r exp.Resolver) error { return exp.ErrNoDefEnv }
func (s *ProjectEnv) Get(sym string) exp.Resolver {
	if sym == "schema" {
		return schemaForm
	}
	prefix := sym[0] == '~'
	if prefix { // strip prefix and continue
		sym = sym[1:]
	}
	split := strings.SplitN(sym, ".", 2)
	if len(split) == 1 { // builtin type
		return nil
	}
	ss := s.Schema(split[0])
	if ss == nil {
		return nil
	}
	return schemaElem(ss, split[1])
}

// SchemaEnv is used inside schema definitions and resolves previously declared models.
type SchemaEnv struct {
	parent exp.Env
	Schema *Schema
}

func (s *SchemaEnv) Parent() exp.Env      { return s.parent }
func (s *SchemaEnv) Supports(x byte) bool { return x == '~' }

func (s *SchemaEnv) Def(sym string, r exp.Resolver) error { return exp.ErrNoDefEnv }

func (r *SchemaEnv) Get(sym string) exp.Resolver {
	if sym[0] == '~' {
		if len(sym) < 3 || sym[1] != '.' {
			return nil
		}
		sym = sym[2:]
	}
	return schemaElem(r.Schema, sym)
}

// ModelEnv wraps a schema env and resolves previously declared fields or constants.
type ModelEnv struct {
	*SchemaEnv
	Model *Model
}

func (r *ModelEnv) Get(sym string) exp.Resolver {
	if sym[0] != '~' {
		r := modelElem(r.Model, sym[1:])
		if r != nil {
			return r
		}
	}
	return r.SchemaEnv.Get(sym)
}

func schemaElem(s *Schema, key string) exp.Resolver {
	split := strings.SplitN(key, ".", 2)
	m := s.Model(split[0])
	if m == nil {
		return nil
	}
	if len(split) > 1 {
		return modelElem(m, split[1])
	}
	return exp.LitResolver{typ.Type{m.Kind, &typ.Info{Ref: m.Ref}}}
}

func modelElem(m *Model, key string) exp.Resolver {
	if m.Kind&typ.MaskPrim != 0 {
		c := m.Const(key)
		if c.Const != nil {
			return exp.LitResolver{constLit(m, c.Const)}
		}
	} else {
		e := m.Field(key)
		if e.Param != nil {
			return exp.TypedUnresolver{e.Type}
		}
	}
	return nil
}

func constLit(m *Model, c *cor.Const) lit.Lit {
	if m.Kind != typ.KindEnum {
		return lit.FlagInt{m.Type, lit.Int(c.Val)}
	}
	return lit.EnumStr{m.Type, lit.Str(strings.ToLower(c.Name))}
}
