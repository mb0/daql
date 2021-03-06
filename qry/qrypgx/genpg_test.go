package qrypgx

import (
	"strings"
	"testing"

	"github.com/mb0/daql/dom/domtest"
	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/typ"
)

func TestGenQuery(t *testing.T) {
	f, err := domtest.ProdFixture()
	if err != nil {
		t.Fatalf("parse prod fixture error: %v", err)
	}
	tests := []struct {
		raw  string
		want []string
	}{
		{`(qry count:#prod.cat)`, []string{`SELECT COUNT(*) FROM prod.cat`}},
		{`(qry *prod.cat)`, []string{
			`SELECT id, name FROM prod.cat`,
		}},
		{`(qry ?prod.cat)`, []string{
			`SELECT id, name FROM prod.cat LIMIT 1`,
		}},
		{`(qry ?prod.cat _:name)`, []string{
			`SELECT name FROM prod.cat LIMIT 1`,
		}},
		{`(qry ?prod.cat off:2)`, []string{
			`SELECT id, name FROM prod.cat LIMIT 1 OFFSET 2`,
		}},
		{`(qry *prod.cat _ id;)`, []string{
			`SELECT id FROM prod.cat`,
		}},
		{`(qry *prod.cat (gt .name 'B'))`, []string{
			`SELECT id, name FROM prod.cat WHERE name > 'B'`,
		}},
		{`(qry *prod.cat asc:name)`, []string{
			`SELECT id, name FROM prod.cat ORDER BY name`,
		}},
		{`(qry *prod.cat _ id; label:('label: ' .name))`, []string{
			`SELECT id, 'label: ' || name FROM prod.cat`,
		}},
		{`(qry ?prod.prod (eq .name 'A') _ name; cname:(?prod.cat (eq .id ..cat) _:name))`,
			[]string{
				`SELECT p.name, c.name FROM prod.prod p, prod.cat c ` +
					`WHERE p.name = 'A' AND c.id = p.cat LIMIT 1`,
			}},
		{`(qry *prod.cat (or (eq .name 'b') (eq .name 'c'))
			+ prods:(#prod.prod (eq .cat ..id))
		)`, []string{
			`SELECT c.id, c.name, ` +
				`(SELECT COUNT(*) FROM prod.prod p WHERE p.cat = c.id) ` +
				`FROM prod.cat c WHERE c.name = 'b' OR c.name = 'c'`,
		}},
		{`(qry *prod.cat (or (eq .name 'b') (eq .name 'c'))
			+ prods:(*prod.prod (eq .cat ..id) _:id)
		)`, []string{
			`SELECT c.id, c.name, ` +
				`(SELECT jsonb_agg(p.id) FROM prod.prod p WHERE p.cat = c.id) ` +
				`FROM prod.cat c WHERE c.name = 'b' OR c.name = 'c'`,
		}},
		{`(qry *prod.cat (or (eq .name 'b') (eq .name 'c'))
			+ prods:(*prod.prod (eq .cat ..id) _ id; name;)
		)`, []string{
			`SELECT c.id, c.name, (SELECT jsonb_agg(_.*) FROM ` +
				`(SELECT p.id, p.name FROM prod.prod p WHERE p.cat = c.id) _) ` +
				`FROM prod.cat c WHERE c.name = 'b' OR c.name = 'c'`,
		}},
		{`(qry ?prod.prod (eq .id 1) _ name; c:(?prod.cat (eq .id ..cat)))`, []string{
			`SELECT p.name, c.id, c.name FROM prod.prod p, prod.cat c ` +
				`WHERE p.id = 1 AND c.id = p.cat LIMIT 1`,
		}},
		{`(qry ?prod.prod (eq .id 1) _ name; c:(?prod.cat (eq .id ..cat) _:name))`, []string{
			`SELECT p.name, c.name FROM prod.prod p, prod.cat c ` +
				`WHERE p.id = 1 AND c.id = p.cat LIMIT 1`,
		}},
	}
	for _, test := range tests {
		env := qry.NewEnv(nil, &f.Project, nil)
		ex, err := exp.Read(strings.NewReader(test.raw))
		if err != nil {
			t.Errorf("parse %s error %+v", test.raw, err)
			continue
		}
		c := exp.NewProg()
		l, err := c.Resl(env, ex, typ.Void)
		if err != nil {
			t.Errorf("resolve %s error %+v", test.raw, err)
			continue
		}
		d := l.(*exp.Atom).Lit.(*exp.Spec).Impl.(*qry.Doc)
		p, err := Analyse(d)
		if err != nil {
			t.Errorf("analyse project: %v", err)
			continue
		}
		qs, err := genQueries(c, env, p)
		if err != nil {
			t.Errorf("gen queries %s: %v", test.raw, err)
			continue
		}
		if len(qs) != len(test.want) {
			t.Errorf("want %d queries got %d", len(test.want), len(qs))
			continue
		}
		for i, got := range qs {
			if got != test.want[i] {
				t.Errorf("for %s\n\twant %s\n\t got %s", test.raw, test.want[i], got)
			}
		}
	}
}

func genQueries(c *exp.Prog, env exp.Env, p *Plan) (res []string, _ error) {
	for _, j := range p.Jobs {
		s, _, err := genQueryStr(c, env, j)
		if err != nil {
			return nil, err
		}
		res = append(res, s)
	}
	return res, nil
}
