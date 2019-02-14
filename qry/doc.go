/*
Package qry extends the xelf language with query expressions to retrieve external resources.

(qry
	(() Selects any one record into a result. Result names must be unique within a qry.
	      Naked selects use the last key part as name. In this case the record key.)
	+ ?prod.cat

	(() Selects the name field of a random category into the result 'name'.)
	+ ?prod.cat.name

	(() Selects can query all records, or use a limit and offset)
	+all *prod.cat
	+top10 *prod.cat :lim 10
	+page3 *prod.cat :off [20 10]

	(() Selects can use a where clause by name string or parameter)
	+named ?prod.cat (eq .name 'name')
	+param ?prod.cat (eq .name $name)

	(() You can count records)
	+mats #prod.cat (eq .kind 'material')

	(() a field mapping can be used to filter or rename fields)
	+infoLabel *label.label (:: .id +label ('Label: ' .name))
	+leanLabel *label.label (:: -tmpl) (() when sel starts with a - decl '.' is implied)

	(() Select nested queries. The '+.' includes all fields. The '..id' refers to the parent id)
	+nest ?prod.cat (eq .name $name) (:: .
		+prods *prod.prod (eq .cat ..id) :ord .name (:: .id .name)
	)
	(() Previous results can be used in the query)
	+top10prods *prod.prod (in .cat /top10/id) :ord .name
)
*/
package qry
