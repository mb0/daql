package qry_test

import (
	"log"
	"testing"

	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/dom/domtest"
	. "github.com/mb0/daql/qry"
	"github.com/mb0/daql/qry/qrymem"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

var (
	prodProj *dom.Project
	memBed   *qrymem.Backend
)

func init() {
	f, err := domtest.ProdFixture()
	if err != nil {
		log.Fatalf("parse prod fixture error: %v", err)
	}
	prodProj = &f.Project
	memBed = &qrymem.Backend{}
	s := f.Schema("prod")
	for _, kl := range f.Fix.List {
		err = memBed.Add(s.Model(kl.Key), kl.Lit.(*lit.List))
		if err != nil {
			log.Fatalf("add %s error: %v", kl.Key, err)
		}
	}
}

var testQry = `(qry
	+cat   ?prod.cat
	+name  ?prod.cat.name
	+all   *prod.cat
	+top10 *prod.cat :lim 10
	+page3 *prod.cat :off 20 :lim 10
	+named ?prod.cat (eq .name 'a')
	+param ?prod.prod (eq .name 'A') :desc .cat :asc .name
	+numc  #prod.prod (eq .cat 3)
	+infoLabel (*prod.label +id +label ('Label: ' .name))
	+leanLabel (*prod.label +tmpl)
	+nest (?prod.cat (eq .name 'a')
		+prods (*prod.prod (eq .cat ..id) :asc .name +id +name)
	)
	+top10prods *prod.prod (in .cat /top10/id) :asc .name
)`

func TestQry(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{`(qry ?prod.cat)`, `{id:25 name:'y'}`},
		{`(qry +count #prod.cat)`, `{count:7}`},
		{`(qry +cat ?prod.cat +count #prod.cat)`, `{cat:{id:25 name:'y'} count:7}`},
		{`(qry ?prod.cat.name)`, `'y'`},
		{`(qry *prod.cat :lim 3)`, `[{id:25 name:'y'} {id:2 name:'b'} {id:3 name:'c'}]`},
		{`(qry *prod.cat :lim 3 :asc .name)`,
			`[{id:1 name:'a'} {id:2 name:'b'} {id:3 name:'c'}]`},
		{`(qry *prod.cat :off 1 :lim 2 :desc .name)`,
			`[{id:25 name:'y'} {id:24 name:'x'}]`},
		{`(qry ?prod.cat (eq .name 'c'))`, `{id:3 name:'c'}`},
		{`(qry (?prod.label +id +label ('Label: ' .name)))`, `{id:1 label:'Label: M'}`},
		{`(qry (*prod.label :off 1 :lim 2 -tmpl))`, `[{id:2 name:'N'} {id:3 name:'O'}]`},
		{`(qry (?prod.cat (eq .name 'c')
			+prods (*prod.prod (eq .cat ..id) :asc .name +id +name)
		))`, `{id:3 name:'c' prods:[{id:1 name:'A'} {id:3 name:'C'}]}`},
		{`(qry (*prod.cat (or (eq .name 'b') (eq .name 'c'))
			+prods (*prod.prod (eq .cat ..id) :asc .name +id +name)
		))`, `[{id:2 name:'b' prods:[{id:2 name:'B'} {id:4 name:'D'}]} ` +
			`{id:3 name:'c' prods:[{id:1 name:'A'} {id:3 name:'C'}]}]`},
		{testQry, ``},
	}
	for _, test := range tests {
		el, err := exp.ParseString(Builtin, test.raw)
		if err != nil {
			t.Errorf("parse %s error %+v", test.raw, err)
			continue
		}
		c := exp.NewCtx(false, true)
		env := NewEnv(nil, prodProj, memBed)
		l, err := c.Resolve(env, el, typ.Void)
		if err != nil {
			t.Errorf("resolve %s error %+v\nUnresolved: %v\nType Context: %v",
				test.raw, err, c.Unres, c.Ctx)
			continue
		}
		spec := l.(*exp.Spec)
		l, err = spec.Resolve(c, env, &exp.Call{Spec: spec, Args: nil}, typ.Void)
		if err != nil {
			t.Errorf("execute %s error %+v\nUnresolved: %v\nType Context: %v",
				test.raw, err, c.Unres, c.Ctx)
			continue
		}
		if test.want == "" {
			continue
		}
		if got := l.String(); got != test.want {
			t.Errorf("want %s got %s", test.want, got)
		}
	}
}
