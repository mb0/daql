package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/mb0/daql/dom"
)

func graph(args []string) error {
	pr, err := project()
	if err != nil {
		return err
	}
	ss := pr.Schemas
	if len(args) > 0 {
		ss, err = filterSchemas(pr, args)
		if err != nil {
			return err
		}
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "digraph %s {\ngraph [rankdir=LR]\n", pr.Name)
	for _, s := range ss {
		fmt.Fprintf(&b, "subgraph cluster_%s {\n", s.Name)
		fmt.Fprintf(&b, "node[shape=record]\ncolor=gray\nlabel=\"%s\"\n", s.Name)
		for _, m := range s.Models {
			fmt.Fprintf(&b, "\"%s\" [label=\"%s\"]\n", m.Qualified(), m.Name)
		}
		fmt.Fprintf(&b, "}\n")
	}
	rels, err := dom.Relate(pr.Project)
	if err != nil {
		return err
	}
	for _, s := range ss {
		for _, m := range s.Models {
			key := m.Qualified()
			rel := rels[key]
			if rel == nil {
				continue
			}
			for _, r := range rel.Out {
				fmt.Fprintf(&b, "\"%s\"->\"%s\"\n", key, r.B.Qualified())
			}
		}
	}
	fmt.Fprintf(&b, "}\n")
	_, err = io.Copy(os.Stdout, &b)
	return err
}
