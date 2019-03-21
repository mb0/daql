package qrypgx

import (
	"strings"

	"github.com/mb0/daql/gen"
	"github.com/mb0/daql/gen/genpg"
	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
)

/*
The naive and simple solution is to execute each query after the other and merge the result in the
end. For SQL backends each database query has in itself a substantial cost. Any SQL backend should
therefor try to optimize to use as few queries as possible and use batching to parallelize when
available.

Newer postgresql versions have great support for JSON Aggregation, which we can reliably use with
as literals already. We could theoretically try to build the complete plan result in one statement
and return one one JSON blob. Instead we should use JSON aggregation on the row level, because the
blob could get rather large and row level streaming has less overhead and other benefits. Most query
interdependencies refer to a single column, we can turn those into nested queries to allow parallel
execution and avoid data preparation and potentially large query parameters.

The backend should take indexes and table size hints into account when filtering, joining or nesting
queries.

TODO
first make simple queries work then add simple join capability
complex joins and sub selects probably have specialized SQL generation functions and are
specifically directed by the plan execer.
*/

func genQuery(b *gen.Ctx, c *exp.Ctx, env exp.Env, t *qry.Task) error {
	q := t.Query
	if q == nil {
		return cor.Errorf("expression tasks not implemented")
	}
	b.WriteString("SELECT ")
	if q.Ref[0] == '#' {
		b.WriteString("COUNT(*)")
	} else {
		for i, s := range q.Sel {
			if i > 0 {
				b.WriteString(", ")
			}
			if s.Query != nil {
				return cor.Errorf("unexpected sub query %v", s)
			} else if s.Expr != nil {
				err := genpg.WriteEl(b, env, s.Expr)
				if err != nil {
					return err
				}
				b.Fmt(" AS %s", strings.ToLower(s.Name))
			} else { // must be table column
				b.WriteString(strings.ToLower(s.Name))
			}
		}
	}
	b.WriteString(" FROM ")
	b.WriteString(strings.ToLower(q.Ref[1:]))
	if len(q.Whr) > 0 {
		b.WriteString(" WHERE ")
		err := genpg.WriteEl(b, env, q.Whr[0])
		if err != nil {
			return err
		}
	}
	if len(q.Ord) > 0 {
		b.WriteString(" ORDER BY ")
		for i, ord := range q.Ord {
			if i > 0 {
				b.WriteString(", ")
			}
			key := ord.Key
			if key[0] == '.' {
				key = key[1:]
			}
			b.WriteString(key)
			if ord.Desc {
				b.WriteString(" DESC")
			}
		}
	}
	if q.Lim > 0 {
		b.Fmt(" LIMIT %d", q.Lim)
	}
	if q.Off > 0 {
		b.Fmt(" OFFSET %d", q.Off)
	}
	return nil
}
