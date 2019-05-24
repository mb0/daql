package main

import (
	"os"

	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/mig"
	"github.com/mb0/xelf/cor"
)

type Project struct {
	mig.History
	mig.Record
}

func project() (*Project, error) {
	h, err := mig.ReadHistory(*dirFlag)
	if err != nil && err != mig.ErrNoHistory {
		return nil, err
	}
	return &Project{h, h.Curr()}, nil
}

func schema(args []string) (*dom.Schema, error) {
	if len(args) == 0 {
		return nil, cor.Errorf("expects schema name or path to a schema file")
	}
	path := args[0]
	// check if path points to a file
	f, err := os.Open(path)
	if err != nil {
		// otherwise try to discover a schema by that name in current project
		pr, err := project()
		if err != nil {
			return nil, cor.Errorf("found no schema file or any project for %s", path)
		}
		s := pr.Schema(path)
		if s == nil {
			return nil, cor.Errorf("found no schema %q in project %s", path, pr.Name)
		}
		return s, nil
	}
	defer f.Close()
	return dom.Execute(dom.NewEnv(dom.Env, &dom.Project{}), f)
}

func gen(args []string) error {
	pr, err := project()
	if err != nil {
		return err
	}
	// and write to configured output dirs
	_ = pr
	return nil
}

func gengo(args []string) error {
	s, err := schema(args)
	if err != nil {
		return err
	}
	// TODO generate based on schema
	_ = s
	// TODO and write to?
	return nil
}

func genpg(args []string) error {
	s, err := schema(args)
	if err != nil {
		return err
	}
	// TODO generate based on schema
	_ = s
	// TODO and write to?
	return nil
}
