/*
Package qry extends the xelf language with an expressions to query external data sources.

The qry expression either takes plain arguments representing one unnamed task, that returns the
result as-is. Or declarations, that each represent a named task and whose results are combined into
an object.

A task can either be a expression that is resolved in the query environment or a query to a data
source. Tasks have names and can be nested to build a query tree. Expression tasks may again have
nested query expressions, those nested queries are always added to the outermost query plan.

There are different queries, that result either in a list of elements, a single element or the
element count. Where element is most often an obj literal or field literal from within an obj.

The query has a reference that identifies the data source. The reference uses a special prefix to
distinguish query types. The reference than usually continues with an unmarked schema reference, but
can also be relative path pointing to another task or even a literal outside the qry environment.

Queries have a selection, that defaults to all fields for obj queries or the scalar element for all
others. This selection can be used to rename, filter out or add additional fields. This can be used
for computed fields like sub queries and should also be able to return object elements from scalar
queries.

(qry
	(() Selects any one record into a result. Result names must be unique within a qry.
	      Naked selects use the last key part as name. In this case the record key.)
	+ ?prod.cat

	(() Selects the name field of a random category into the result 'name'.)
	+ ?prod.cat.name

	(() Selects can query all records, or use a limit and offset)
	+all   *prod.cat
	+top10 *prod.cat :lim 10
	+page3 *prod.cat :off [20 10]

	(() Selects can use a where clause by name string or parameter)
	+named ?prod.cat (eq .name 'name')
	+param ?prod.cat (eq .name $name)

	(() You can count records)
	+mats #prod.cat (eq .kind 'material')

	(() a field mapping can be used to filter or rename fields)
	+infoLabel *label.label (:: +id +label ('Label: ' .name))
	+leanLabel *label.label (:: -tmpl (() when sub starts with a - decl '+' is implied))

	(() Select nested queries. The '+' includes all fields. The '..id' refers to the parent id)
	(+nest ?prod.cat (eq .name $name) :: +
		+prods *prod.prod (eq .cat ..id) :asc .name (:: +id +name)
	)
	(() Previous results can be used in the query)
	+top10prods *prod.prod (in .cat /top10/id) :asc .name
)
*/
package qry
