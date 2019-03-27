// +build ignore

package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/gen"
	"github.com/mb0/daql/gen/gengo"
	"github.com/mb0/daql/gen/genpg"
	"github.com/mb0/xelf/bfr"
)

func main() {
	flag.Parse()
	fname := flag.Arg(0)
	f, err := os.Open(fname)
	if err != nil {
		log.Fatalf("open file %s error: %v", fname, err)
	}
	defer f.Close()
	pr := &dom.Project{}
	env := dom.NewEnv(dom.Env, pr)
	s, err := dom.Execute(env, f)
	if err != nil {
		log.Fatalf("execute %s error: %v", fname, err)
	}
	writeGo(pr, s)
	writeSql(pr, s)
}
func writeGo(pr *dom.Project, s *dom.Schema) {
	var buf bytes.Buffer
	b := &gen.Ctx{
		Ctx:     bfr.Ctx{B: &buf, Tab: "\t"},
		Project: pr,
		Pkg:     "github.com/mb0/daql/evt",
		Pkgs: map[string]string{
			"cor": "github.com/mb0/xelf/cor",
			"lit": "github.com/mb0/xelf/lit",
			"exp": "github.com/mb0/xelf/exp",
			"evt": "github.com/mb0/daql/evt",
		},
		Header: "// generated code\n\n",
	}
	err := gengo.WriteFile(b, s)
	if err != nil {
		log.Fatalf("gen file evt.go error: %v", err)
	}
	err = ioutil.WriteFile("evt.go", buf.Bytes(), 0644)
	if err != nil {
		log.Fatalf("write evt.go error: %v", err)
	}
}
func writeSql(pr *dom.Project, s *dom.Schema) {
	var buf bytes.Buffer
	b := &gen.Ctx{
		Ctx:     bfr.Ctx{B: &buf, Tab: "\t"},
		Project: pr,
		Pkg:     "evt",
		Header:  "-- generated code\n\n",
	}
	ss := *s
	ss.Models = make([]*dom.Model, 0, len(s.Models))
	for _, m := range s.Models {
		_, backup := m.Extra["backup"]
		if !backup {
			continue
		}
		ss.Models = append(ss.Models, m)
	}
	err := genpg.WriteFile(b, &ss)
	if err != nil {
		log.Fatalf("gen file evt.sql error: %v", err)
	}
	err = ioutil.WriteFile("evt.sql", buf.Bytes(), 0644)
	if err != nil {
		log.Fatalf("write evt.sql error: %v", err)
	}
}
