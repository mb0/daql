package genpg

import (
	"strings"
	"testing"

	"github.com/mb0/daql/gen"
	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/std"
	"github.com/mb0/xelf/typ"
)

func TestRender(t *testing.T) {
	tests := []struct {
		el   string
		want string
	}{
		{`null`, `NULL`},
		{`true`, `TRUE`},
		{`false`, `FALSE`},
		{`23`, `23`},
		{`-42`, `-42`},
		{`'test'`, `'test'`},
		{`(raw 'test')`, `'test'::bytea`},
		{`(uuid '4d85fc61-398b-4886-a396-b67b6453e431')`,
			`'4d85fc61-398b-4886-a396-b67b6453e431'::uuid`},
		{`(time '2019-02-11')`, `'2019-02-11T00:00:00+01:00'::timestamptz`},
		{`(span '1h5m')`, `'1:05:00'::interval`},
		{`[null true]`, `'[null,true]'::jsonb`},
		{`(list|int [1 2 3])`, `'{1,2,3}'::int8[]`},
		{`(list|str ['a' 'b' "'"])`, `'{"a","b","''"}'::text[]`},
		{`{a: null b: true}`, `'{"a":null,"b":true}'::jsonb`},
		{`(or a b)`, `a OR b`},
		{`(not a b)`, `NOT a AND NOT b`},
		{`(and x v)`, `x != 0 AND v != ''`},
		{`(eq x y 1)`, `x = y AND x = 1`},
		{`(equal x 1)`, `(x = 1 AND pg_typeof(x) = pg_typeof(1))`},
		{`(gt x y 1)`, `x > y AND y > 1`},
		{`(if a x 1)`, `CASE WHEN a THEN x ELSE 1 END`},
		{`(add (add x 2) 3)`, `x + 2 + 3`},
		{`(add (mul x 2) 3)`, `x * 2 + 3`},
		{`(add 3 (mul x 2))`, `3 + x * 2`},
		{`(mul (add x 2) 3)`, `(x + 2) * 3`},
		{`(mul 3 (add x 2))`, `3 * (x + 2)`},
		{`(and (or a b) c)`, `(a OR b) AND c`},
		{`(or (and a b) c)`, `a AND b OR c`},
	}
	env := exp.NewScope(exp.Builtin{std.Core})
	unresed(env, typ.Bool, "a", "b", "c")
	unresed(env, typ.Str, "v", "w")
	unresed(env, typ.Int, "x", "y")
	for _, test := range tests {
		ex, err := exp.ParseString(env, test.el)
		if err != nil {
			t.Errorf("parse %s err: %v", test.el, err)
			continue
		}
		el, err := exp.Resolve(env, ex)
		if err != nil && err != exp.ErrUnres {
			t.Errorf("resolve %s err: %v", test.el, err)
			continue
		}
		var b strings.Builder
		err = WriteEl(&gen.Ctx{Ctx: bfr.Ctx{B: &b}}, env, el)
		if err != nil {
			t.Errorf("render %s err: %v", test.el, err)
			continue
		}
		got := b.String()
		if got != test.want {
			t.Errorf("%s want %s got %s", el, test.want, got)
		}
	}
}

func unresed(env *exp.Scope, t typ.Type, names ...string) {
	for _, n := range names {
		env.Def(n, &exp.Def{Type: t})
	}
}
