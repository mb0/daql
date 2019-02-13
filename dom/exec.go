package dom

import (
	"fmt"
	"io"
	"strings"

	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lex"
	"github.com/mb0/xelf/lit"
)

func ExecuteString(env exp.Env, s string) (*Schema, error) {
	return Execute(env, strings.NewReader(s))
}

func Execute(env exp.Env, r io.Reader) (*Schema, error) {
	t, err := lex.New(r).Scan()
	if err != nil {
		return nil, err
	}
	x, err := exp.Parse(t)
	if err != nil {
		return nil, err
	}
	c := &exp.Ctx{Exec: true}
	l, err := c.Resolve(env, x)
	if err != nil {
		return nil, fmt.Errorf("%v %s", err, c.Unres)
	}
	s, ok := getPtr(l).(*Schema)
	if !ok {
		return nil, fmt.Errorf("expected *dom.Schema got %T", r)
	}
	return s, nil
}

func getPtr(e exp.El) interface{} {
	if a, ok := e.(lit.Assignable); ok {
		return a.Ptr()
	}
	return nil
}
