package mig

import "github.com/mb0/daql/dom"

// Record consists of a project definition and its manifest recorded at a point in time.
type Record struct {
	Manifest
	*dom.Project
}

// History contains records of one project.
type History struct {
	Records []Record
}
