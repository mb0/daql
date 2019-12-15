package qry

import (
	"strings"

	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/std"
	"github.com/mb0/xelf/typ"
	"github.com/mb0/xelf/utl"
)

var Builtin = exp.Builtin{
	utl.StrLib.Lookup(),
	utl.TimeLib.Lookup(),
	std.Core, std.Decl,
}

// QryEnv provide the qry form and the required facilities for resolving and executing queries.
type QryEnv struct {
	Project *dom.ProjectEnv
	Backend Backend
}

func NewEnv(env exp.Env, pr *dom.Project, bend Backend) *QryEnv {
	if env == nil {
		env = Builtin
	}
	return &QryEnv{dom.NewEnv(env, pr), bend}
}

func FindEnv(env exp.Env) *QryEnv {
	for env != nil {
		env = exp.Supports(env, '?')
		if p, ok := env.(*QryEnv); ok {
			return p
		}
		if env != nil {
			env = env.Parent()
		}
	}
	return nil
}

func (qe *QryEnv) Parent() exp.Env      { return qe.Project }
func (qe *QryEnv) Supports(x byte) bool { return x == '?' }
func (qe *QryEnv) Get(sym string) *exp.Def {
	if sym == "qry" {
		return exp.NewDef(qrySpec)
	}
	return nil
}

func (qe *QryEnv) Qry(q string, arg lit.Lit) (lit.Lit, error) {
	el, err := exp.Read(strings.NewReader(q))
	if err != nil {
		return nil, cor.Errorf("parse qry %s error: %w", q, err)
	}
	if arg == nil {
		arg = lit.Nil
	}
	d := &exp.Dyn{Els: []exp.El{el, &exp.Atom{Lit: arg}}}
	l, err := exp.Eval(qe, d)
	if err != nil {
		return nil, cor.Errorf("eval qry %s error: %w", el, err)
	}
	if a, ok := l.(*exp.Atom); ok {
		return a.Lit, nil
	}
	return nil, cor.Errorf("qry result %T is not an atom", l)
}

// TaskInfo holds task details during query execution.
// Done indicates whether the task and all its sub task are represented by data.
type TaskInfo struct {
	Data lit.Proxy
	Done bool
}

type DocEnv struct {
	Par   exp.Env
	Doc   *Doc
	Data  lit.Proxy
	Infos map[*Task]TaskInfo
}

func (de *DocEnv) Parent() exp.Env      { return de.Par }
func (de *DocEnv) Supports(x byte) bool { return x == '/' }
func (de *DocEnv) Get(sym string) *exp.Def {
	if sym[0] != '/' {
		return nil
	}
	if len(sym) == 1 {
		return &exp.Def{Type: de.Doc.Type, Lit: de.Data}
	}
	t, path, err := RootTask(de.Doc, sym)
	if err != nil {
		return nil
	}
	var data lit.Lit
	if de.Data == nil {
		data = t.Type
	} else {
		n := de.Infos[t]
		if !n.Done {
			return nil
		}
		data = n.Data
	}
	l, err := lit.SelectPath(data, path)
	if err != nil {
		return nil
	}
	if de.Data == nil {
		return &exp.Def{Type: l.(typ.Type)}
	}
	return exp.NewDef(l)
}

func (d *Doc) ReslEnv(par exp.Env) *DocEnv {
	return &DocEnv{par, d, nil, nil}
}

func (d *Doc) EvalEnv(par exp.Env) *DocEnv {
	t, opt := d.Type.Deopt()
	data := lit.ZeroProxy(t)
	if opt {
		data = lit.SomeProxy{data}
	}
	return &DocEnv{par, d, data, make(map[*Task]TaskInfo, len(d.Root)*3)}
}

func (de *DocEnv) Prep(parent lit.Proxy, t *Task) (lit.Proxy, error) {
	pp := lit.Deopt(parent)
	k, ok := pp.(lit.Keyer)
	if !ok || t.Name == "" {
		return parent, nil
	}
	l, err := k.Key(cor.Keyed(t.Name))
	if err != nil {
		return nil, err
	}
	p, ok := l.(lit.Proxy)
	if !ok {
		return nil, cor.Errorf("prep task result for %s want proxy got %T", t.Name, l)
	}
	return p, nil
}

func (de *DocEnv) Done(t *Task, data lit.Proxy) {
	n := de.Infos[t]
	n.Data = data
	n.Done = true
	de.Infos[t] = n
}

type TaskEnv struct {
	Par exp.Env
	Doc *DocEnv
	*Task
	Param lit.Lit
}

func (te *TaskEnv) Parent() exp.Env      { return te.Par }
func (te *TaskEnv) Supports(x byte) bool { return x == '.' }
func (te *TaskEnv) Get(sym string) *exp.Def {
	if sym[0] != '.' {
		return nil
	}
	sym = sym[1:]
	if te.Query != nil {
		if te.Param != nil {
			l, err := lit.Select(te.Param, sym)
			if err == nil {
				return exp.NewDef(l)
			}
		}
		for _, t := range te.Query.Sel {
			if t.Name != sym {
				continue
			}
			if te.Doc != nil {
				nfo := te.Doc.Infos[t]
				if nfo.Done {
					return exp.NewDef(nfo.Data)
				}
			}
			return &exp.Def{Type: t.Type}
		}
		// otherwise check query result type
		p, _, err := te.Query.Type.ParamByKey(sym)
		if err == nil {
			return &exp.Def{Type: p.Type}
		}
	}
	if te.Task.Parent != nil {
		p, _, err := te.Task.Parent.Type.ParamByKey(sym)
		if err == nil {
			return &exp.Def{Type: p.Type}
		}
		if te.Task.Parent.Query != nil {
			p, _, err = te.Task.Parent.Query.Type.ParamByKey(sym)
			if err == nil {
				return &exp.Def{Type: p.Type}
			}
		}
	}
	// resolves to result from result type
	p, _, err := te.Type.ParamByKey(sym)
	if err == nil {
		return &exp.Def{Type: p.Type}
	}
	return nil
}

type SelEnv struct {
	Par exp.Env
	*Task
}

func (se *SelEnv) Parent() exp.Env      { return se.Par }
func (se *SelEnv) Supports(x byte) bool { return x == '.' }
func (se *SelEnv) Get(sym string) *exp.Def {
	if sym[0] != '.' {
		return nil
	}
	sym = sym[1:]
	// resolves to result from query type
	p, _, err := se.Query.Type.ParamByKey(sym)
	if err == nil || err == exp.ErrUnres {
		return &exp.Def{Type: p.Type}
	}
	// otherwise check previous selection
	for _, t := range se.Query.Sel {
		if t.Name != sym {
			continue
		}
		return &exp.Def{Type: t.Type}
	}
	return nil
}
