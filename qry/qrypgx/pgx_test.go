package qrypgx

import (
	"testing"

	"github.com/jackc/pgx"
	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/dom/domtest"
	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/typ"
)

const dsn = `host=/var/run/postgresql dbname=daql`

func TestPgx(t *testing.T) {
	f, err := domtest.ProdFixture()
	if err != nil {
		t.Fatalf("parse prod fixture error: %v", err)
	}
	domEnv := dom.NewEnv(qry.Builtin, &f.Project)
	db, err := Open(dsn, nil)
	if err != nil {
		t.Fatalf("parse prod fixture error: %v", err)
	}
	defer setup(t, db, &f.Project)()
	err = CopyFrom(db, f.Schema("prod"), f.ProdFix)
	if err != nil {
		t.Fatalf("copy fixtures error: %v", err)
	}
	tests := []struct {
		raw  string
		want string
	}{
		{"(qry #prod.cat)", "7"},
		{"(qry ?prod.cat :asc .name)", "{id:1 name:'a'}"},
		{"(qry *prod.cat (lt .id 3) :asc .name)",
			"[{id:1 name:'a'} {id:2 name:'b'}]"},
	}
	pgxBed := New(db, &f.Project)
	for _, test := range tests {
		el, err := exp.ParseString(test.raw)
		if err != nil {
			t.Errorf("parse %s error %+v", test.raw, err)
			continue
		}
		env := qry.NewEnv(domEnv, pgxBed)
		c := &exp.Ctx{Exec: true}
		l, err := c.Resolve(env, el, typ.Void)
		if err != nil {
			t.Errorf("resolve %s error %+v\n%v", el, err, c.Unres)
			continue
		}
		if got := l.String(); got != test.want {
			t.Errorf("want %s got %s", test.want, got)
		}
	}
}

func setup(t *testing.T, db *pgx.ConnPool, p *dom.Project) func() {
	err := CreateProject(db, p)
	if err != nil {
		t.Fatalf("create project err: %v", err)
	}
	return func() {
		/*
			err := DropProject(db, p)
			if err != nil {
				t.Errorf("drop schema err %v", err)
			}*/
	}
}
