/*
Package qry extends the xelf language with a form to construct query functions for external data.

The qry form constructs a query plan for a configured model project. The form either takes plain
arguments representing one unnamed task, or declarations, that each represent a named task and whose
results are combined into a record literal. A query plan, once resolved, can be cached and reused.

The plan's root tasks can either be query or expression tasks. Each query task has a selection, that
consists of one or more tasks. The query selection tasks can again be expressions or sub queries.
Selections without an explicit query or expression, act as a reference to the subject field of the
same name.

The query has a reference that identifies the data source. The reference uses special prefixes to
distinguish query types. The reference then usually continues with an unmarked schema reference, but
can also be relative path pointing to another task or even a literal outside the qry environment.
Three different query prefixes indicate, whether the result is a list of elements, a single element
or the element count.

The following paragraphs are planned and not yet implemented.

The plan acts as a function spec that takes one record parameter and returns the plan's result. This
allows users to use inline query evaluation, when the situation calls for it.

Expression task can reference previous results or contain nested query constructors or executions.
This would allow conditional queries or post processing of query results.

(qry
	(() Selects any one record into a result. Result names must be unique within a qry.
	      Naked selects use the last key part as name. In this case the record key.)
        cat:?prod.cat

	(() Selects the name field of a random category into the result 'name'.)
	name:(?prod.cat _:name)

	(() Selects can query all records, or use a limit and offset)
	all:*prod.cat
	top10:(*prod.cat lim:10)
	page3:(*prod.cat off:20 lim:10)

	(() Selects can use a where clause by name string or parameter)
	named:(?prod.cat (eq .name 'name'))
	param:(?prod.cat (eq .name $name))

	(() You can count records)
	mats:(#prod.cat (eq .kind 'material'))

	(() a field mapping can be used to filter or rename fields)
	infoLabel:(*label.label _ id; label:('Label: ' .name))
	leanLabel:(*label.label - tmpl;) (() when sub starts with a - symbol '+' is implied))

	(() Select nested queries. The '+' includes all fields. The '..id' refers to the parent id)
	nest:(?prod.cat (eq .name $name) + prods:(*prod.prod (eq .cat ..id) asc:name _ id; name;))
	(() Previous results can be used in the query)
	top10prods:(*prod.prod (in .cat /top10/id) asc:name)
)

We could construct a query from tagged struct types, that will also be populated with the results.

type MyQuery struct {
	All   []prod.Cat `qry:"*prod.cat"`
	Top10 []prod.Cat `qry:"*prod.cat lim:10"`
	Mats  int        `qry:"#prod.cat (eq .kind 'material')"`
	Nest  *struct{
		prod.Cat `qry:"+"`
		Prods []struct{
			ID   [16]byte
			Name string
		} `qry:"*prod.prod (eq .cat ..id)" asc:.name`
	} `qry:"?prod.cat (eq .name $name)"`
}

TODO think about simplifying the query processing by relying more on the exp primitives, instead of
the current declarative approach. We already thought about nested queries; what if queries are just
any xelf scripts with nested queries. A query symbol, currently a ref, would resolve to a query
spec. We would resolve the root element and collect all queries, plan and then execute. The plan may
organize query calls in multiple batches. The execution however uses the default process, and calls
the backend for each query. The backend will execute a batch when the first of its queries is
requested and caches the results for the other batched queries.

This would resolve to a query spec that stores the query ref:
	*prod.cat

Starting an expression expands to a call:
	(*prod.cat)

And can be used in any expression:
	(div (add (#prod.cat) 9) 10)

A query doc could in the end be just be a record constructor:
	(sel
	all:(*prod.cat)
	top10:(*prod.cat lim:10)
	page3:(*prod.cat off:20 lim:10)
	t10p: (*prod.prod (in .cat ..top10/id) asc:.name)

	infoLabel: (*label.label (:id; label:('Label: ' .name)))
	leanLabel: (*label.label (:-tmpl;))

	(() Select nested queries. The '+' includes all fields. The '..id' refers to the parent id)
	nest:(?prod.cat (eq .name $name) (:+
		prods:(*prod.prod (eq .cat ..id) asc:.name (:id; name;))
	))
	(() Previous results can be used in the query)
	top10prods: (*prod.prod (in .cat /top10/id) asc:.name)
	)

This would require a partially resolved dot scope for the sel resolver or calls in general.
Absolute paths may generally refer to the whole program result in any xelf script. After resolution
we know the program result and can create the proxy and pass it into the evaluation.

This would make querying data more powerful for complex queries and flexible for use in other
context like the repl or templates.

Just like the whr expression takes a query subject value as dot scope and returns a boolean, with a
sel expression we could remove decls from the syntax and handle it like an expression called with
a subject value dot and returning a literal. The default selection would be an identity function
returning its input. The forms to restrict or extend record type would generally applicable and
useful in other situations.

A scalar queries would be more regular (*prod.cat (:.name))

(with . (sel a; b; c:(add 1 .c)))
*/
package qry
