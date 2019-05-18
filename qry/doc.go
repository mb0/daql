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
	+ ?prod.cat

	(() Selects the name field of a random category into the result 'name'.)
	+ ?prod.cat.name

	(() Selects can query all records, or use a limit and offset)
	+all   *prod.cat
	+top10 *prod.cat :lim 10
	+page3 *prod.cat :off 20 :lim 10

	(() Selects can use a where clause by name string or parameter)
	+named ?prod.cat (eq .name 'name')
	+param ?prod.cat (eq .name $name)

	(() You can count records)
	+mats #prod.cat (eq .kind 'material')

	(() a field mapping can be used to filter or rename fields)
	(+infoLabel *label.label +id +label ('Label: ' .name))
	(+leanLabel *label.label -tmpl (() when sub starts with a - decl '+' is implied))

	(() Select nested queries. The '+' includes all fields. The '..id' refers to the parent id)
	(+nest ?prod.cat (eq .name $name)
		+ +prods *prod.prod (eq .cat ..id) :asc .name (:: +id +name)
	)
	(() Previous results can be used in the query)
	+top10prods *prod.prod (in .cat /top10/id) :asc .name
)

We could construct a query from tagged struct types, that will also be populated with the results.

type MyQuery struct {
	All   []prod.Cat `qry:"*prod.cat"`
	Top10 []prod.Cat `qry:"*prod.cat :lim 10"`
	Mats  int        `qry:"#prod.cat (eq .kind 'material')"`
	Nest  *struct{
		prod.Cat `qry:"."`
		Prods []struct{
			ID   [16]byte
			Name string
		} `qry:"*prod.prod (eq .cat ..id)" :asc .name`
	} `qry:"?prod.cat (eq .name $name)"`
}
*/
package qry
