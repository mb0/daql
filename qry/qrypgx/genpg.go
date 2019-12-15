package qrypgx

import (
	"strings"

	"github.com/mb0/daql/gen/genpg"
	"github.com/mb0/daql/qry"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/typ"
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
 * query parameters
 * batching
 * replace query dependencies with id-sub-queries
 * more tests for complex joins and inline queries
*/

func genQueryStr(c *exp.Prog, env exp.Env, j *Job) (string, []genpg.Param, error) {
	var sb strings.Builder
	w := genpg.NewWriter(&sb, jobTranslator{})
	err := genSelect(w, c, env, j)
	if err != nil {
		return "", nil, err
	}
	return sb.String(), w.Params, nil
}

func genSelect(w *genpg.Writer, c *exp.Prog, env exp.Env, j *Job) error {
	prefix := j.Kind&(KindJoin|KindInline) != 0 || j.Parent != nil
	w.WriteString("SELECT ")
	if j.IsScalar() {
		if j.Kind&KindCount != 0 {
			w.WriteString("COUNT(*)")
		} else if j.Kind&KindJSON != 0 {
			w.WriteString("jsonb_agg(")
			if prefix {
				w.WriteString(j.Alias[j.Task])
				w.WriteByte('.')
			}
			w.WriteString(j.Cols[0].Name)
			w.WriteByte(')')
		} else {
			if prefix {
				w.WriteString(j.Alias[j.Task])
				w.WriteByte('.')
			}
			w.WriteString(j.Cols[0].Name)
		}
	} else {
		jenv := &jobEnv{Alias: j.Alias, Task: j.Task, Env: env, Prefix: prefix}
		if j.Kind&KindJSON != 0 {
			w.WriteString("jsonb_agg(_.*) FROM (SELECT ")
		}
		for i, col := range j.Cols {
			if i > 0 {
				w.WriteString(", ")
			}
			if col.Job.Kind&KindInlined != 0 && col.Job.Parent == j {
				w.WriteByte('(')
				err := genSelect(w, c, env, col.Job)
				if err != nil {
					return err
				}
				w.WriteByte(')')
				continue
			}
			if col.Expr != nil {
				err := w.WriteEl(jenv, col.Expr)
				if err != nil {
					return err
				}
				continue
			}
			if prefix {
				w.WriteString(j.Alias[col.Job.Task])
				w.WriteByte('.')
			}
			w.WriteString(col.Key)
		}
		if j.Kind&KindJSON != 0 {
			defer w.WriteString(") _")
		}
	}
	w.WriteString(" FROM ")
	whr := make([]*qry.Task, 0, len(j.Tabs))
	for i, tab := range j.Tabs {
		if i > 0 {
			w.WriteString(", ")
		}
		w.WriteString(getTableName(tab.Query))
		if prefix {
			w.WriteByte(' ')
			w.WriteString(j.Alias[tab])
		}
		if tab.Query.Whr != nil {
			whr = append(whr, tab)
		}
	}
	if len(whr) != 0 {
		w.WriteString(" WHERE ")
		for i, e := range whr {
			if i > 0 {
				w.WriteString(" AND ")
			}
			wenv := &jobEnv{Alias: j.Alias, Task: e, Env: env, Prefix: prefix}
			el, err := c.Resl(wenv, e.Query.Whr, typ.Void)
			if err != nil && err != exp.ErrVoid && err != exp.ErrUnres {
				return err
			}
			err = w.WriteEl(wenv, el)
			if err != nil {
				return err
			}
		}
	}
	return genQueryCommon(w, j.Query)
}

func genQueryCommon(b *genpg.Writer, q *qry.Query) error {
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
