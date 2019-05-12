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
	l, err := c.Resolve(env, x, typ.Void)
	if err != nil {
		return nil, cor.Errorf("%s: %v", c.Unres, err)
	}
	s, ok := getPtr(l).(*Schema)
	if !ok {
		return nil, cor.Errorf("expected *dom.Schema got %T", r)
	}
	return s, nil
}

func getPtr(e exp.El) interface{} {
	if a, ok := e.(lit.Assignable); ok {
		return a.Ptr()
	}
	return nil
}
