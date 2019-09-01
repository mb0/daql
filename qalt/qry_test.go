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
		{`(#prod.cat :off 5 :lim 5)`, `2`},
		{`(?prod.cat)`, `{id:25 name:'y'}`},
		{`(?prod.cat - +id)`, `{id:25}`},
		{`(?prod.cat/id)`, `25`},
		{`(?prod.cat :off 1)`, `{id:2 name:'b'}`},
		{`(*prod.cat :lim 2)`, `[{id:25 name:'y'} {id:2 name:'b'}]`},
		{`(*prod.cat :asc .name :lim 2)`, `[{id:1 name:'a'} {id:2 name:'b'}]`},
		{`(*prod.prod :desc .cat :asc .name :lim 3)`,
			`[{id:1 name:'A' cat:3} {id:3 name:'C' cat:3} {id:2 name:'B' cat:2}]`},
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
