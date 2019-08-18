package dom

import (
	"io"
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

func ExecuteString(env exp.Env, s string) (*Schema, error) {
	return Execute(env, strings.NewReader(s))
}

func Execute(env exp.Env, r io.Reader) (*Schema, error) {
	x, err := exp.Read(r)
	if err != nil {
		return nil, err
	}
	c := exp.NewCtx()
	x, err = c.Resl(env, x, typ.Void)
	if err != nil {
		return nil, err
	}
	a, err := c.Eval(env, x, typ.Void)
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
	x, err := exp.Read(r)
	if err != nil {
		return err
	}
	if env == nil {
		env = Env
	}
	_, err = exp.NewCtx().WithPart(true).Eval(NewEnv(env, pr), x, typ.Void)
	return err
}
