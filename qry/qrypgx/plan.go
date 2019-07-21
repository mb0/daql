package qrypgx

import (
	"strings"

	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
)

type Kind uint

const (
	KindMulti = 1 << iota
	KindSingle
	KindCount
	KindScalar
	KindJoin
	KindJoined
	KindInline
	KindInlined
	KindJSON
)

func (k Kind) IsMulti() bool  { return k&KindMulti != 0 }
func (k Kind) IsSingle() bool { return k&KindSingle != 0 }
func (k Kind) IsScalar() bool { return k&KindScalar != 0 }
func (k Kind) IsJoined() bool { return k&KindJoined != 0 }

// Job augments a task with additional information and collects nested and joined jobs.
type Job struct {
	Kind
	// Task is the primary subject of this job.
	*qry.Task
	Cols   []Column
	Tabs   []*qry.Task
	Alias  map[*qry.Task]string
	Parent *Job
	// Deps is the list of immediate dependency tasks for this job, except the parent.
	Deps []*qry.Task
}

func (j *Job) DependsOn(t *qry.Task) bool {
	for _, d := range j.Deps {
		if d == t {
			return true
		}
	}
	return false
}

type Column struct {
	*qry.Task
	Key string
	Job *Job
}

type Plan struct {
	*qry.Doc
	Jobs []*Job
}

// Analyse populates the dependencies for each task in the query plan or returns an error.
func Analyse(d *qry.Doc) (*Plan, error) {
	p := &Plan{Doc: d}
	// TODO sort into batches as part of the analysis process
	for _, t := range d.Root {
		a := newAliaser()
		j, err := p.newJob(t, nil, a)
		if err != nil {
			return nil, err
		}
		j.Tabs = append(j.Tabs, j.Task)
		// root tasks are always distinct jobs, for now
		p.Jobs = append(p.Jobs, j)
		if t.Query == nil {
			err = p.exprDeps(j, t.Expr)
			if err != nil {
				return nil, err
			}
		} else {
			err = p.analyseQuery(j, a)
			if err != nil {
				return nil, err
			}
		}
	}
	return p, nil
}

func (p *Plan) newJob(t *qry.Task, par *Job, a aliaser) (*Job, error) {
	j := &Job{Task: t, Alias: a.alias, Parent: par}
	if t.Query == nil {
		return j, nil
	}
	a.addAlias(j.Task)
	s := strings.SplitN(t.Query.Ref[1:], ".", 3)
	if len(s) < 2 {
		return nil, cor.Errorf("unqualified query %s", t.Query.Ref)
	}
	if len(s) > 2 {
		j.Kind |= KindScalar
	}
	switch t.Query.Ref[0] {
	case '#':
		j.Kind |= KindCount | KindScalar
	case '?':
		j.Kind |= KindSingle
	case '*':
		j.Kind |= KindMulti
	}
	return j, nil
}

func (p *Plan) analyseQuery(j *Job, a aliaser) error {
	err := p.exprDeps(j, j.Query.Whr)
	if err != nil {
		return err
	}
	for _, t := range j.Query.Sel {
		if t.Query == nil {
			col := Column{Task: t, Job: j, Key: strings.ToLower(t.Name)}
			if t.Expr != nil {
				err := p.exprDeps(j, t.Expr)
				if err != nil {
					return err
				}
			}
			j.Cols = append(j.Cols, col)
			continue
		}
		s, err := p.newJob(t, j, a)
		if err != nil {
			return err
		}
		err = p.analyseQuery(s, a)
		if err != nil {
			return err
		}
		if !s.IsSingle() || !s.DependsOn(j.Task) {
			j.Kind |= KindInline
			s.Kind |= KindInlined
			if s.Kind&KindMulti != 0 {
				s.Kind |= KindJSON
			}
			s.Tabs = append(s.Tabs, t)
			col := Column{Task: t, Job: s, Key: strings.ToLower(t.Name)}
			j.Cols = append(j.Cols, col)
			continue
		}
		err = p.addJoin(j, s)
		if err != nil {
			return err
		}
	}
	return nil
}
func (p *Plan) addJoin(j, s *Job) error {
	j.Kind |= KindJoin
	s.Kind |= KindJoined
	j.Tabs = append(j.Tabs, s.Task)
	if s.IsScalar() {
		col := Column{Task: s.Task, Job: s, Key: getScalarName(s.Query)}
		j.Cols = append(j.Cols, col)
		return nil
	}
	j.Cols = append(j.Cols, s.Cols...)
	s.Cols = nil
	return nil
}

func (p *Plan) exprDeps(j *Job, x exp.El) error {
	if x == nil {
		return nil
	}
	return x.Traverse(&depVisitor{Plan: p, Job: j})
}

// TODO implement nested query expressions
type depVisitor struct {
	exp.Ghost
	*Plan
	*Job
}

func (v *depVisitor) VisitSym(s *exp.Sym) error {
	switch s.Name[0] {
	case '/':
		t, _, err := qry.RootTask(v.Doc, s.Name)
		if err != nil {
			return err
		}
		v.Deps = append(v.Deps, t)
	case '.':
		n := s.Name[1:]
		if n == "" || n[0] != '.' {
			return nil
		}
		t := v.Task
		for n != "" && n[0] == '.' {
			t = t.Parent
			if t == nil {
				return cor.Errorf("no task for relative symbol %s", s.Name)
			}
			n = n[1:]
		}
		v.Deps = append(v.Deps, t)
	}
	return nil
}

type aliaser struct {
	keys  map[string]struct{}
	alias map[*qry.Task]string
}

func newAliaser() aliaser {
	return aliaser{make(map[string]struct{}), make(map[*qry.Task]string)}
}

func (a aliaser) addAlias(t *qry.Task) string {
	const digits = "1234567890"
	n := getModelName(t.Query)
	for _, k := range [...]string{n[:1], n} {
		if a.try(k, t) {
			return k
		}
		for i := 0; i < 10; i++ {
			if k1 := k + digits[i:i+1]; a.try(k1, t) {
				return k1
			}
		}
	}
	return "FAIL"
}

func (a aliaser) try(k string, t *qry.Task) bool {
	if _, ok := a.keys[k]; !ok {
		a.keys[k] = struct{}{}
		a.alias[t] = k
		return true
	}
	return false
}

func getScalarName(q *qry.Query) string {
	s := strings.SplitN(q.Ref[1:], ".", 3)
	if len(s) < 3 {
		return ""
	}
	return s[2]
}
func getTableName(q *qry.Query) string {
	n := q.Ref[1:]
	s := strings.SplitN(n, ".", 3)
	if len(s) < 3 {
		return n
	}
	return n[:len(s[0])+1+len(s[1])]
}
func getModelName(q *qry.Query) string {
	s := strings.Split(q.Ref[1:], ".")
	if len(s) > 1 {
		return s[1]
	}
	return s[0]
}
