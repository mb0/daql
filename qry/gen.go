// +build ignore

package main

import (
	"flag"
	"log"

	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/gen"
	"github.com/mb0/daql/gen/gengo"
)

func main() {
	flag.Parse()
	fname, pr := flag.Arg(0), &dom.Project{}
	s, err := gen.DomFile(flag.Arg(0), pr)
	if err != nil {
		log.Fatalf("dom file %s error: %v", fname, err)
	}
	writeGo(pr, s)
}

func writeGo(pr *dom.Project, s *dom.Schema) {
	b := &gen.Ctx{
		Project: pr,
		Pkg:     "github.com/mb0/daql/qry",
		Pkgs: map[string]string{
			"cor": "github.com/mb0/xelf/cor",
			"lit": "github.com/mb0/xelf/lit",
			"typ": "github.com/mb0/xelf/typ",
			"exp": "github.com/mb0/xelf/exp",
			"qry": "github.com/mb0/daql/qry",
		},
		Header: "// generated code\n\n",
	}
	err := gengo.WriteFile(b, "qry_gen.go", s)
	if err != nil {
		log.Fatalf("write file error: %v", err)
	}
}
