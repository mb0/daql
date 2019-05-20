// Package auth provides common tools and interfaces for user authentication and authorization.
package auth

import (
	"sync"

	"github.com/mb0/xelf/cor"
	"golang.org/x/crypto/bcrypt"
)

type Signer interface {
	Sign(pass string) (string, error)
	Verify(token, pass string) error
}

type Store interface {
	Save(user, token string) error
	Token(user string) (string, error)
}

type Tokens struct {
	sync.RWMutex
	toks map[string]string
}

func (t *Tokens) Save(user, token string) error {
	t.Lock()
	defer t.Unlock()
	if t.toks == nil {
		t.toks = make(map[string]string)
	}
	t.toks[user] = token
	return nil
}

func (t *Tokens) Token(user string) (string, error) {
	t.RLock()
	defer t.RUnlock()
	token, ok := t.toks[user]
	if !ok {
		return token, cor.Errorf("no token found for user %s", user)
	}
	return token, nil
}

type Bcrypt struct {
	Cost int
}

func (v *Bcrypt) Sign(pass string) (string, error) {
	token, err := bcrypt.GenerateFromPassword([]byte(pass), v.Cost)
	if err != nil {
		return "", err
	}
	return string(token), nil
}

func (v *Bcrypt) Verify(token, pass string) error {
	return bcrypt.CompareHashAndPassword([]byte(token), []byte(pass))
}
