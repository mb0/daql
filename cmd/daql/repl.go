package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

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
	readReplHistory(lin)
	lin.SetMultiLineMode(true)
	var buf bytes.Buffer
	var multi bool
	env := &exp.DataScope{Par: qry.Builtin, Def: exp.Def{Lit: &lit.Dict{}}}
	for {
		prompt := "> "
		if multi = buf.Len() > 0; multi {
			prompt = "… "
		}
		got, err := lin.Prompt(prompt)
		if err != nil {
			buf.Reset()
			if err == io.EOF {
				writeReplHistory(lin)
				fmt.Println()
				return nil
			}
			log.Printf("unexpected error reading prompt: %v", err)
			continue
		}
		got = strings.TrimSpace(got)
		if got == "" {
			continue
		}
		if multi {
			buf.WriteByte(' ')
		}
		buf.WriteString(got)
		el, err := exp.Read(&buf)
		if err != nil {
			if cor.IsErr(err, io.EOF) {
				continue
			}
			buf.Reset()
			log.Printf("error parsing %s: %v", got, err)
			continue
		}
		lin.AppendHistory(buf.String())
		buf.Reset()
		qenv := qry.NewEnv(env, &fix.Project, membed)
		l, err := exp.Eval(qenv, el)
		if err != nil {
			log.Printf("error resolving %s: %v", got, err)
			continue
		}
		fmt.Printf("= %s\n\n", l)
	}
	return nil
}

func replHistoryPath() string {
	path, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(path, "daql/repl.history")
}

func readReplHistory(lin *liner.State) {
	path := replHistoryPath()
	if path == "" {
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	_, err = lin.ReadHistory(f)
	if err != nil {
		log.Printf("error reading repl history file %q: %v\n", path, err)
	}
}

func writeReplHistory(lin *liner.State) {
	path := replHistoryPath()
	if path == "" {
		return
	}
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		log.Printf("error creating dir for repl history %q: %v\n", dir, err)
		return
	}
	f, err := os.Create(path)
	if err != nil {
		log.Printf("error creating file for repl history %q: %v\n", path, err)
		return
	}
	defer f.Close()
	_, err = lin.WriteHistory(f)
	if err != nil {
		log.Printf("error writing repl history file %q: %v\n", path, err)
	}
}
