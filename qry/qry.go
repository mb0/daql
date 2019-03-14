package qry

import (
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

type Plan struct {
	Root []*Task
	// Result will be populated with the full query result. It indicates the expected result
	// type and if set to a proxy literal, results are assigned directly to it.
	Result lit.Assignable
	Simple bool
}

type Task struct {
	Name  string
	Expr  exp.El
	Query *Query

	// Type is the task's result type or void if not yet resolved.
	Type typ.Type
	// Result will be populated with the full task result. It indicates the expected result
	// type and if set to a proxy literal, results are assigned directly to it.
	Result lit.Assignable
	Done   bool
}

type Ord struct {
	Key  string
	Desc bool
}

type Query struct {
	Ref string
	// Type represents the query subject type.
	Type typ.Type
	// Whr is a list of expression elements treated as 'and' arguments.
	// The whr clause can only refer to full subject but none of the extra selections.
	Whr exp.Dyn
	// Order is a list of result symbols used for ordering. We may at some point allow
	// references to the subject, like sql does in the ordering clause.
	Ord []Ord
	Off int
	Lim int
	Sel []*Task
}

type Backend interface {
	ExecPlan(*exp.Ctx, exp.Env, *Plan) error
}
