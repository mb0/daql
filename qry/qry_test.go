package qry_test

import (
	"log"
	"testing"

	"github.com/mb0/daql/dom"
	. "github.com/mb0/daql/qry"
	"github.com/mb0/daql/qry/qrymem"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/typ"
)

var (
	domEnv exp.Env
	memBed *qrymem.Backend
)

type Cat struct {
	ID   int
	Name string
}

type Prod struct {
	ID   int
	Name string
	Cat  int
}

type Label struct {
	ID   int
	Name string
	Tmpl []byte
}

func init() {
	schema := `(schema 'prod'
		(+Cat   +ID int :pk +Name str)
		(+Prod  +ID int :pk +Name str +Cat  int)
		(+Label	+ID int :pk +Name str +Tmpl raw)
	)`
	domEnv = dom.NewEnv(Builtin, &dom.Project{})
	s, err := dom.ExecuteString(domEnv, schema)
	if err != nil {
		log.Fatalf("parse test schema error: %v", err)
	}
	memBed = &qrymem.Backend{}
	err = memBed.Add(s.Model("cat"), &[]Cat{
		{25, "y"},
		{2, "b"},
		{3, "c"},
		{1, "a"},
		{4, "d"},
		{26, "z"},
		{24, "x"},
	})
	if err != nil {
		log.Fatalf("add cats error: %v", err)
	}
	err = memBed.Add(s.Model("prod"), &[]Prod{
		{25, "Y", 1},
		{2, "B", 2},
		{3, "C", 3},
		{1, "A", 3},
		{4, "D", 2},
		{26, "Z", 1},
	})
	if err != nil {
		log.Fatalf("add cats error: %v", err)
	}
	err = memBed.Add(s.Model("label"), &[]Label{
		{1, "M", []byte("foo")},
		{2, "N", []byte("bar")},
		{3, "O", []byte("spam")},
		{3, "P", []byte("egg")},
	})
	if err != nil {
		log.Fatalf("add cats error: %v", err)
	}
}

var testQry = `(qry
	+cat ?prod.cat
	+name ?prod.cat.name
	+all   *prod.cat
	+top10 *prod.cat :lim 10
	+page3 *prod.cat :off 20 :lim 10
	+named ?prod.cat (eq .name 'a')
	+param ?prod.prod (eq .name 'A') :desc cat :asc .name
	+numc  #prod.prod (eq cat 3)
	+infoLabel *prod.label (:: +id +label ('Label: ' .name))
	+leanLabel *prod.label (:: +tmpl)
	+nest ?prod.cat (eq .name 'a') (::
		+prods *prod.prod (eq .cat ..id) :asc .name (:: +id +name)
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
		{`(qry ?prod.cat.name)`, `'y'`},
		{`(qry *prod.cat :lim 3)`, `[{id:25 name:'y'} {id:2 name:'b'} {id:3 name:'c'}]`},
		{`(qry *prod.cat :lim 3 :asc .name)`,
			`[{id:1 name:'a'} {id:2 name:'b'} {id:3 name:'c'}]`},
		{`(qry *prod.cat :off 1 :lim 2 :desc .name)`,
			`[{id:25 name:'y'} {id:24 name:'x'}]`},
		{`(qry ?prod.cat (eq .name 'c'))`, `{id:3 name:'c'}`},
		{`(qry ?prod.label (:: +id +label ('Label: ' .name)))`, `{id:1 label:'Label: M'}`},
		{`(qry *prod.label :off 1 :lim 2 (:: -tmpl))`, `[{id:2 name:'N'} {id:3 name:'O'}]`},
		{`(qry ?prod.cat (eq .name 'c') (::
			+prods *prod.prod (eq .cat ..id) :asc .name (:: +id +name)
		))`, `{id:3 name:'c' prods:[{id:1 name:'A'} {id:3 name:'C'}]}`},
		{`(qry *prod.cat (or (eq .name 'b') (eq .name 'c')) (::
			+prods *prod.prod (eq .cat ..id) :asc .name (:: +id +name)
		))`, `[{id:2 name:'b' prods:[{id:2 name:'B'} {id:4 name:'D'}]} ` +
			`{id:3 name:'c' prods:[{id:1 name:'A'} {id:3 name:'C'}]}]`},
		{testQry, ``},
	}
	for _, test := range tests {
		el, err := exp.ParseString(test.raw)
		if err != nil {
			t.Errorf("parse %s error %+v", test.raw, err)
			continue
		}
		c := &exp.Ctx{}
		env := NewEnv(domEnv, memBed)
		l, err := c.Resolve(env, el, typ.Void)
		if err != nil {
			t.Errorf("resolve %s error %+v\n%v", el, err, c.Unres)
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
