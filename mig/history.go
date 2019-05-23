package mig

import "github.com/mb0/daql/dom"

type Record struct {
	dom.Manifest
	*dom.Project
}

type History struct {
	Records []Record
}
