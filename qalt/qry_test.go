package qry_test

import (
	"log"
	"testing"

	"github.com/mb0/daql/dom/domtest"
	. "github.com/mb0/daql/qalt"
	"github.com/mb0/daql/qry"

	"github.com/mb0/xelf/lit"
)

func getBackend() Backend {
	f := domtest.Must(domtest.ProdFixture())
	b := &LitBackend{Project: &f.Project, Data: make(map[string]*lit.List)}
	s := f.Schema("prod")
	for _, kl := range f.Fix.List {
		err := b.Add(s.Model(kl.Key), kl.Lit.(*lit.List))
		if err != nil {
			log.Printf("test backend error: %v", err)
		}
	}
	return b
}

func TestSubj(t *testing.T) {
	b := getBackend()
	tests := []struct {
		Raw  string
		Want string
	}{
		{`(#prod.cat)`, `7`},
		{`(#prod.prod)`, `6`},
		{`([] (#prod.cat) (#prod.prod))`, `[7 6]`},
		{`({} cats:(#prod.cat) prods:(#prod.prod))`, `{cats:7 prods:6}`},
		{`(#prod.cat off:5 lim:5)`, `2`},
		{`(?prod.cat)`, `{id:25 name:'y'}`},
		{`(?prod.cat (eq .id 1) _:name)`, `'a'`},
		{`(?prod.cat (eq .name 'a'))`, `{id:1 name:'a'}`},
		{`(?prod.cat _ id;)`, `{id:25}`},
		{`(?prod.cat _:id)`, `25`},
		{`(?prod.cat off:1)`, `{id:2 name:'b'}`},
		{`(*prod.cat lim:2)`, `[{id:25 name:'y'} {id:2 name:'b'}]`},
		{`(*prod.cat asc:name off:1 lim:2)`, `[{id:2 name:'b'} {id:3 name:'c'}]`},
		{`(*prod.cat desc:name lim:2)`, `[{id:26 name:'z'} {id:25 name:'y'}]`},
		{`(?prod.label _ id; label:('Label: ' .name))`, `{id:1 label:'Label: M'}`},
		{`(*prod.label off:1 lim:2 - tmpl;)`, `[{id:2 name:'N'} {id:3 name:'O'}]`},
		{`(*prod.prod desc:cat asc:name lim:3)`,
			`[{id:1 name:'A' cat:3} {id:3 name:'C' cat:3} {id:2 name:'B' cat:2}]`},
		{`(?prod.cat (eq .name 'c') +
			prods:(*prod.prod (eq .cat ..id) asc:name _ id; name;)
		)`, `{id:3 name:'c' prods:[{id:1 name:'A'} {id:3 name:'C'}]}`},
		{`(*prod.cat (or (eq .name 'b') (eq .name 'c')) asc:name +
			prods:(*prod.prod (eq .cat ..id) asc:name _ id; name;)
		)`, `[{id:2 name:'b' prods:[{id:2 name:'B'} {id:4 name:'D'}]} ` +
			`{id:3 name:'c' prods:[{id:1 name:'A'} {id:3 name:'C'}]}]`},
	}
	env := NewEnv(qry.Builtin, b)
	for _, test := range tests {
		el, err := env.Qry(test.Raw, nil)
		if err != nil {
			t.Errorf("qry %s error %+v", test.Raw, err)
			continue
		}
		if got := el.String(); got != test.Want {
			t.Errorf("want for %s\n\t%s got %s", test.Raw, test.Want, got)
		}
	}
}
