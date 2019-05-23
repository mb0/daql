package main

import (
	"fmt"
	"io"
	"log"

	"github.com/mb0/daql/dom/domtest"
	"github.com/mb0/daql/qry"
	"github.com/mb0/daql/qry/qrymem"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/lit"
	"github.com/peterh/liner"
)

func repl(args []string) error {
	// use fixture and memory backend for now
	fix, err := domtest.ProdFixture()
	if err != nil {
		return cor.Errorf("parse fixture: %v", err)
	}
	membed := &qrymem.Backend{}
	prodsch := fix.Schema("prod")
	for _, kl := range fix.Fix.List {
		err = membed.Add(prodsch.Model(kl.Key), kl.Lit.(*lit.List))
		if err != nil {
			return cor.Errorf("prepare backend, add %s: %v", kl.Key, err)
		}
	}
	// TODO use the backup and a temporary database if we have a dataset argument
	// otherwise try the configured db
	lin := liner.NewLiner()
	defer lin.Close()
	lin.SetMultiLineMode(true)
	var got string
	for i := 0; ; i++ {
		if i == 0 {
			got, err = lin.PromptWithSuggestion("> ", "(qry )", 5)
		} else {
			got, err = lin.Prompt("> ")
		}
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				return nil
			}
			log.Printf("unexpected error reading prompt: %v", err)
			continue
		}
		el, err := exp.ParseString(qry.Builtin, got)
		if err != nil {
			log.Printf("error parsing %s: %v", got, err)
			continue
		}
		lin.AppendHistory(got)
		l, err := exp.Execute(qry.NewEnv(qry.Builtin, &fix.Project, membed), el)
		if err != nil {
			log.Printf("error resolving %s: %v", got, err)
			continue
		}
		fmt.Printf("= %s\n\n", l)
	}
	return nil
}
