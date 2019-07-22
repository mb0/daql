package dom

import (
	"io"
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lex"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

func ExecuteString(env exp.Env, s string) (*Schema, error) {
	return Execute(env, strings.NewReader(s))
}

func Execute(env exp.Env, r io.Reader) (*Schema, error) {
	t, err := lex.New(r).Scan()
	if err != nil {
		return nil, err
	}
	x, err := exp.Parse(env, t)
	if err != nil {
		return nil, err
	}
	c := exp.NewCtx(true, true)
	a, err := c.Resolve(env, x, typ.Void)
	if err != nil {
		return nil, cor.Errorf("%s: %v", c.Unres, err)
	}
	s, ok := getPtr(a).(*Schema)
	if !ok {
		return nil, cor.Errorf("expected *dom.Schema got %s", a)
	}
	return s, nil
}

func getPtr(e exp.El) interface{} {
	if a, ok := e.(*exp.Atom); ok {
		if p, ok := a.Lit.(lit.Proxy); ok {
			return p.Ptr()
		}
	}
	return nil
}

func Read(r io.Reader, env exp.Env, pr *Project) error {
	t, err := lex.New(r).Scan()
	if err != nil {
		return err
	}
	x, err := exp.Parse(env, t)
	if err != nil {
		return err
	}
	if env == nil {
		env = Env
	}
	_, err = exp.NewCtx(true, true).Resolve(NewEnv(env, pr), x, typ.Void)
	return err
}
