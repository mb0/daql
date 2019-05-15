package domtest

import (
	"testing"
)

func TestDomtest(t *testing.T) {
	_, err := ProdFixture()
	if err != nil {
		t.Fatalf("prod fixture error: %v", err)
	}
	_, err = PersonFixture()
	if err != nil {
		t.Fatalf("person fixture error: %v", err)
	}
}
