// Package pol provides a simple role based access control system.
package pol

import "github.com/mb0/xelf/cor"

// Policy allows users to execute an action or returns an error.
type Policy interface {
	Allow(user, action string) error
}

// Rules implements a role base policy.
type Rules struct{ roles map[string]*role }

func NewPolicy(def bool) *Rules { return &Rules{roles: make(map[string]*role)} }

func (p *Rules) AddRole(name string, def bool) *Rules {
	p.role(name).def = def
	return p
}
func (p *Rules) AddMember(role, group string) *Rules {
	s := p.role(role)
	s.roles = append(s.roles, p.role(group))
	return p
}
func (p *Rules) Allow(role, action string) *Rules {
	s := p.role(role)
	s.allow = append(s.allow, action)
	return p
}
func (p *Rules) Deny(role, action string) *Rules {
	s := p.role(role)
	s.deny = append(s.deny, action)
	return p
}

func (p *Rules) Police(user, action string) error {
	s := p.roles[user]
	if s == nil {
		return cor.Errorf("subject %q is unknown", user)
	}
	if !s.def && !s.allowed(action) {
		return cor.Errorf("subject %q is not allowed to %q", user, action)
	}
	if s.denied(action) {
		return cor.Errorf("subject %q is denied to %q", user, action)
	}
	return nil
}

func (p *Rules) role(sub string) (s *role) {
	if s = p.roles[sub]; s == nil {
		s = &role{name: sub}
		p.roles[sub] = s
	}
	return s
}

type role struct {
	name  string
	def   bool
	allow []string
	deny  []string
	roles []*role
}

func (s *role) allowed(act string) bool {
	for _, a := range s.allow {
		if act == a {
			return true
		}
	}
	for _, r := range s.roles {
		if r.allowed(act) {
			return true
		}
	}
	return false
}

func (s *role) denied(act string) bool {
	for _, a := range s.deny {
		if act == a {
			return true
		}
	}
	for _, r := range s.roles {
		if r.denied(act) {
			return true
		}
	}
	return false
}
