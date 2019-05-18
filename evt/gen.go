// +build ignore

package main

import (
	"flag"
	"log"

	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/gen"
	"github.com/mb0/daql/gen/gengo"
	"github.com/mb0/daql/gen/genpg"
)

func main() {
	flag.Parse()
	fname, pr := flag.Arg(0), &dom.Project{}
	s, err := gen.DomFile(flag.Arg(0), pr)
	if err != nil {
		log.Fatalf("dom file %s error: %v", fname, err)
	}
	writeGo(pr, s)
	writeSql(pr, s)
}

func writeGo(pr *dom.Project, s *dom.Schema) {
	b := gengo.NewCtx(pr, "evt", "github.com/mb0/daql/evt")
	err := gengo.WriteFile(b, "evt.go", s)
	if err != nil {
		log.Fatalf("write file error: %v", err)
	}
}

func writeSql(pr *dom.Project, s *dom.Schema) {
	b := &gen.Ctx{
		Project: pr,
		Pkg:     "evt",
		Header:  "-- generated code\n\n",
	}
	ss := *s
	ss.Models = make([]*dom.Model, 0, len(s.Models))
	for _, m := range s.Models {
		b, _ := m.Extra.Key("backup")
		if b.IsZero() {
			continue
		}
		ss.Models = append(ss.Models, m)
	}
	err := genpg.WriteFile(b, "evt.sql", &ss)
	if err != nil {
		log.Fatalf("write file error: %v", err)
	}
}
