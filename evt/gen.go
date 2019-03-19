// +build ignore

package main

import (
	"bufio"
	"flag"
	"log"
	"os"

	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/gen"
	"github.com/mb0/daql/gen/gengo"
	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/exp"
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
	out := bufio.NewWriter(os.Stdout)
	b := &gen.Ctx{
		Ctx: bfr.Ctx{B: out},
		Pkg: "github.com/mb0/daql/evt",
		Pkgs: map[string]string{
			"cor": "github.com/mb0/xelf/cor",
			"evt": "github.com/mb0/daql/evt",
		},
		Header: "// generated code\n\n",
	}
	var els []exp.El
	for _, m := range s.Models {
		els = append(els, m.Typ())
	}
	err = gengo.WriteFile(b, els)
	if err != nil {
		log.Fatalf("write file %s error: %v", fname, err)
	}
	err = out.Flush()
	if err != nil {
		log.Fatalf("flush file %s error: %v", fname, err)
	}
}
