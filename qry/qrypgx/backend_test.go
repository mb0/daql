package qrypgx

import (
	"testing"

	"github.com/jackc/pgx"
	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/dom/domtest"
	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/lit"
)

const dsn = `host=/var/run/postgresql dbname=daql`

func TestBackend(t *testing.T) {
	f, err := domtest.ProdFixture()
	if err != nil {
		t.Fatalf("parse prod fixture error: %v", err)
	}
	db, err := Open(dsn, nil)
	if err != nil {
		t.Fatalf("parse prod fixture error: %v", err)
	}
	defer setup(t, db, &f.Project)()
	err = CopyFrom(db, f.Schema("prod"), f.Fix)
	if err != nil {
		t.Fatalf("copy fixtures error: %v", err)
	}
	tests := []struct {
		Raw, Want string
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
	}
	b := New(db, &f.Project)
	arg := lit.RecFromKeyed([]lit.Keyed{{"a", lit.Int(1)}})
	env := qry.NewEnv(nil, &f.Project, b)
	for _, test := range tests {
		l, err := env.Qry(test.Raw, arg)
		if err != nil {
			t.Errorf("query %s error %+v", test.Raw, err)
			continue
		}
		if got := l.String(); got != test.Want {
			t.Errorf("want %s got %s", test.Want, got)
		}
	}
}

func setup(t *testing.T, db *pgx.ConnPool, p *dom.Project) func() {
	err := CreateProject(db, p)
	if err != nil {
		t.Fatalf("create project err: %v", err)
	}
	return func() {
		err := DropProject(db, p)
		if err != nil {
			t.Errorf("drop schema err %v", err)
		}
	}
}
