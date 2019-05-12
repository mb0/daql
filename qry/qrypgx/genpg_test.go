package qrypgx

import (
	"strings"
	"testing"

	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/dom/domtest"
	"github.com/mb0/daql/gen"
	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/typ"
)

func TestGenQuery(t *testing.T) {
	f, err := domtest.ProdFixture()
	if err != nil {
		t.Fatalf("parse prod fixture error: %v", err)
	}
	denv := dom.NewEnv(qry.Builtin, &f.Project)
	tests := []struct {
		raw  string
		want string
	}{
		{`(qry #prod.cat)`, `SELECT COUNT(*) FROM prod.cat`},
		{`(qry *prod.cat)`, `SELECT id, name FROM prod.cat`},
		{`(qry ?prod.cat)`, `SELECT id, name FROM prod.cat LIMIT 1`},
		{`(qry ?prod.cat :off 2)`, `SELECT id, name FROM prod.cat LIMIT 1 OFFSET 2`},
		{`(qry (*prod.cat +id))`, `SELECT id FROM prod.cat`},
		{`(qry *prod.cat (gt .name 'B'))`,
			`SELECT id, name FROM prod.cat WHERE name > 'B'`},
		{`(qry *prod.cat :asc .name)`, `SELECT id, name FROM prod.cat ORDER BY name`},
		{`(qry (*prod.cat +id +label ('label: ' .name)))`,
			`SELECT id, 'label: ' || name AS label FROM prod.cat`},
	}
	for _, test := range tests {
		ex, err := exp.ParseString(denv, test.raw)
		if err != nil {
			t.Errorf("parse %s error %+v", test.raw, err)
			continue
		}
		env := qry.NewEnv(denv, nil)
		c := exp.NewCtx(true, false)
		_, err = c.Resolve(env, ex, typ.Void)
		if err != nil && err != exp.ErrExec {
			t.Errorf("resolve %s error %+v", test.raw, err)
			continue
		}
		p := env.Plan
		if !p.Simple {
			t.Errorf("expecting simple query %s", test.raw)
			continue
		}
		var buf strings.Builder
		err = genQuery(&gen.Ctx{Ctx: bfr.Ctx{B: &buf}}, c, env, p.Root[0])
		if err != nil {
			t.Errorf("gen query %s error %+v", test.raw, err)
			continue
		}
		got := buf.String()
		if got != test.want {
			t.Errorf("for %s want %s got %s", test.raw, test.want, got)
		}
	}
}
