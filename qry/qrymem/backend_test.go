package qrymem

import (
	"log"
	"testing"

	"github.com/mb0/daql/dom/domtest"
	"github.com/mb0/daql/mig"
	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/lit"
)

func getBackend() *Backend {
	f := domtest.Must(domtest.ProdFixture())
	b := &Backend{Record: mig.Record{Project: &f.Project}}
	s := f.Schema("prod")
	for _, kl := range f.Fix.List {
		err := b.Add(s.Model(kl.Key), kl.Lit.(*lit.List))
		if err != nil {
			log.Fatalf("add %s error: %v", kl.Key, err)
		}
	}
	return b
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
	+leanLabel (*prod.label -tmpl)
	+nest (?prod.cat (eq .name 'a')
		+prods (*prod.prod (eq .cat ..id) :asc .name +id +name)
	)
	(() TODO
	+top10prods *prod.prod (in .cat /top10/id) :asc .name
	)
)`

func TestBackend(t *testing.T) {
	b := getBackend()
	tests := []struct {
		Raw  string
		Want string
	}{
		{`(qry ?prod.cat)`, `{id:25 name:'y'}`},
		{`(qry +count #prod.cat)`, `{count:7}`},
		{`(qry +cat ?prod.cat +count #prod.cat)`, `{cat:{id:25 name:'y'} count:7}`},
		{`(qry ?prod.cat (eq .id 1))`, `{id:1 name:'a'}`},
		{`(qry ?prod.cat (eq .id $a))`, `{id:1 name:'a'}`},
		{`(qry ?prod.cat.name (eq .id 1))`, `'a'`},
		{`(qry *prod.cat :off 1 :lim 2 :asc .name)`, `[{id:2 name:'b'} {id:3 name:'c'}]`},
		{`(qry *prod.cat :lim 2 :desc .name)`, `[{id:26 name:'z'} {id:25 name:'y'}]`},
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
		{`(qry *prod.cat :off 1 :lim 2 :desc .name)`,
			`[{id:25 name:'y'} {id:24 name:'x'}]`},
		{`(qry +cat ?prod.cat :asc .name +count #prod.cat)`,
			`{cat:{id:1 name:'a'} count:7}`},
		{`(qry (?prod.prod (eq .id 1) +name +c (?prod.cat (eq .id ..cat))))`,
			`{name:'A' c:{id:3 name:'c'}}`},
		{`(qry (?prod.prod (eq .id 1) +name +cn (?prod.cat.name (eq .id ..cat))))`,
			`{name:'A' cn:'c'}`},
		{testQry, ``},
	}
	arg := lit.RecFromKeyed([]lit.Keyed{{"a", lit.Int(1)}})
	env := qry.NewEnv(qry.Builtin, b.Project, b)
	for _, test := range tests {
		l, err := env.Qry(test.Raw, arg)
		if err != nil {
			t.Errorf("query %s error %+v", test.Raw, err)
			continue
		}
		if test.Want == "" {
			continue
		}
		if got := l.String(); got != test.Want {
			t.Errorf("want for %s\n\t%s got %s", test.Raw, test.Want, got)
		}
	}
}
