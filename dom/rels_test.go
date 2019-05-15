package dom_test

import (
	"testing"

	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/dom/domtest"
)

func TestRelate(t *testing.T) {
	tests := []struct {
		fix *domtest.Proj
	}{
		{domtest.Must(domtest.ProdFixture())},
		{domtest.Must(domtest.PersonFixture())},
	}
	for _, test := range tests {
		rels, err := dom.Relate(&test.fix.Project)
		if err != nil {
			t.Errorf("relate err: %v", err)
			continue
		}
		t.Logf("rels for %s %s", test.fix.Name, rels)
		if len(rels) == 0 {
			t.Errorf("no rels found")
		}
	}
}
