package domtest

import (
	"time"
)

const PersonRaw = `(schema 'person'
	(+Group  +ID int :pk +Name str)
	(+Contact +Addr str)
	(+Person +ID int :pk +Name str +Family int :ref '..Group' + @Contact?)
	(+Member +ID int :pk
		+Person int :ref '..Person'
		+Group  int :ref '..Group'
		+Joined time
	)
)`

type Group struct {
	ID   int
	Name string
}

type Person struct {
	ID     int
	Name   string
	Family int
}

type Member struct {
	ID     int
	Person int
	Group  int
	Joined time.Time
}

const PersonFixRaw = `{
	group:[
		[1  'Schnabels']
		[2  'Starkeys']
		[3  'Beatles']
		[4  'Gophers']
	]
	person:[
		[1  'Martin' 1 {addr: 'baumstr. 23'}]
		[2  'Ringo'  2 null]
		[3  'Rob'    0 null]
	]
	member:[
		[1 1 1 '1983-11-07']
		[2 2 2 '1940-07-07']
		[3 2 3 '1962-08-01']
		[4 1 4 '2012-02-20']
		[5 3 4 '2009-10-10']
	]
}`

func PersonFixture() (*Proj, error) { return Fixture(PersonRaw, PersonFixRaw) }
