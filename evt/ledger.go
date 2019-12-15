package evt

import (
	"time"

	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
)

// NextRev returns a rev truncated to ms or if rev is not after last the next possible revision one
// millisecond after the last.
func NextRev(last, rev time.Time) time.Time {
	rev = rev.Truncate(time.Millisecond)
	if rev.After(last) {
		return rev
	}
	return last.Add(time.Millisecond)
}

// Ledger abstracts over the event storage. It allows to access the latest revision and query
// events. Ledger implemetations are usually not thread-safe unless explicitly documented.
type Ledger interface {
	// Rev returns the latest event revision or the zero time.
	Rev() time.Time
	Project() *dom.Project
	// Events returns the ledger events filtered by the given expression and parameters.
	Events(whr exp.Dyn, param lit.Lit) ([]*Event, error)
}

// Publisher is a ledger that can publish transactions.
type Publisher interface {
	Ledger
	Publish(Trans) ([]*Event, error)
}

// Replicator is a ledger that can replicate events.
type Replicator interface {
	Ledger
	Replicate([]*Event) error
}
