package domtest

import (
	"log"
	"testing"
)

func TestDomtest(t *testing.T) {
	_, err := ProdFixture()
	if err != nil {
		log.Fatalf("prod fixture error: %v", err)
	}
}
