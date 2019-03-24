package hub

import (
	"regexp"
	"strings"
)

// RouterFunc implements Router for simple route functions.
type RouterFunc func(*Msg)

func (r RouterFunc) Route(m *Msg) { r(m) }

// MatchFilter only routes messages, that match one of a list of subjects.
type MatchFilter struct {
	Router
	Match []string
}

// NewMatchFilter returns a new filtered router r that exact-matches strs.
func NewMatchFilter(r Router, strs ...string) *MatchFilter {
	return &MatchFilter{r, strs}
}

func (r *MatchFilter) Route(m *Msg) {
	for _, s := range r.Match {
		if m.Subj == s {
			r.Router.Route(m)
			return
		}
	}
}

// PrefixFilter only routes messages, that match one of a list of subject prefixes.
type PrefixFilter struct {
	Router
	Prefix []string
}

// NewPrefixFilter returns a new filtered router r that prefix-matches strs.
func NewPrefixFilter(r Router, strs ...string) *PrefixFilter {
	return &PrefixFilter{r, strs}
}

func (r *PrefixFilter) Route(m *Msg) {
	for _, s := range r.Prefix {
		if strings.HasPrefix(m.Subj, s) {
			r.Router.Route(m)
			return
		}
	}
}

// RegexpFilter only routes messages, which subjects match a regular expression.
type RegexpFilter struct {
	Router
	*regexp.Regexp
}

func (r *RegexpFilter) Route(m *Msg) {
	if r.Regexp == nil || r.Regexp.MatchString(m.Subj) {
		r.Router.Route(m)
	}
}

// Routers is a slice of routers, all of them are called with incoming messages.
type Routers []Router

func (rs Routers) Route(m *Msg) {
	for _, r := range rs {
		r.Route(m)
	}
}
