package qrypgx

import (
	"strings"

	"github.com/mb0/daql/gen"
	"github.com/mb0/daql/gen/genpg"
	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/bfr"
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

func genQueryStr(c *exp.Ctx, env exp.Env, j *Job) (_ string, err error) {
	var sb strings.Builder
	b := &gen.Ctx{Ctx: bfr.Ctx{B: &sb}}
	err = genSelect(b, c, env, j)
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}

func genSelect(b *gen.Ctx, c *exp.Ctx, env exp.Env, j *Job) error {
	prefix := j.Kind&(KindJoin|KindInline) != 0 || j.Parent != nil
	b.WriteString("SELECT ")
	if j.IsScalar() {
		sc := getScalarName(j.Query)
		if j.Kind&KindCount != 0 {
			if sc == "" {
				sc = "*"
			}
			b.WriteString("COUNT(")
			b.WriteString(sc)
			b.WriteByte(')')
		} else if j.Kind&KindJSON != 0 {
			b.WriteString("jsonb_agg(")
			if prefix {
				b.WriteString(j.Alias[j.Task])
				b.WriteByte('.')
			}
			b.WriteString(sc)
			b.WriteByte(')')
		} else {
			if prefix {
				b.WriteString(j.Alias[j.Task])
				b.WriteByte('.')
			}
			b.WriteString(sc)
		}
	} else {
		jenv := &jobEnv{Job: j, Task: j.Task, Env: env, Prefix: prefix}
		if j.Kind&KindJSON != 0 {
			b.WriteString("jsonb_agg(_.*) FROM (SELECT ")
		}
		for i, col := range j.Cols {
			if i > 0 {
				b.WriteString(", ")
			}
			if col.Job.Kind&KindInlined != 0 && col.Job.Parent == j {
				b.WriteByte('(')
				err := genSelect(b, c, env, col.Job)
				if err != nil {
					return err
				}
				b.WriteByte(')')
				continue
			}
			if col.Expr != nil {
				err := genpg.WriteEl(b, jenv, col.Expr)
				if err != nil {
					return err
				}
				continue
			}
			if prefix {
				b.WriteString(j.Alias[col.Job.Task])
				b.WriteByte('.')
			}
			b.WriteString(col.Key)
		}
		if j.Kind&KindJSON != 0 {
			defer b.WriteString(") _")
		}
	}
	b.WriteString(" FROM ")
	whr := make([]*qry.Task, 0, len(j.Tabs))
	for i, tab := range j.Tabs {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(getTableName(tab.Query))
		if prefix {
			b.WriteByte(' ')
			b.WriteString(j.Alias[tab])
		}
		if len(tab.Query.Whr.Els) > 0 {
			whr = append(whr, tab)
		}
	}
	if len(whr) != 0 {
		b.WriteString(" WHERE ")
		for i, w := range whr {
			if i > 0 {
				b.WriteString(" AND ")
			}
			wenv := &jobEnv{Job: j, Task: w, Env: env, Prefix: prefix}
			err := genpg.WriteEl(b, wenv, w.Query.Whr.Els[0])
			if err != nil {
				return err
			}
		}
	}
	return genQueryCommon(b, j.Query)
}

func genQueryCommon(b *gen.Ctx, q *qry.Query) error {
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
