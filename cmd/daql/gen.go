package main

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/gen"
	"github.com/mb0/daql/gen/gengo"
	"github.com/mb0/daql/gen/genpg"
	"github.com/mb0/daql/mig"
	"github.com/mb0/xelf/cor"
)

func generate(args []string) error {
	pr, err := project()
	if err != nil {
		return err
	}
	ss := pr.Schemas
	for _, gf := range genfuncs {
		err = gf.gen(pr, ss)
		if err != nil {
			return err
		}
	}
	return nil
}

var genfuncs = []struct {
	key string
	gen func(*Project, []*dom.Schema) error
}{
	{"go", gogen},
	{"pg", pggen},
}

func genXX(xx string, args []string) error {
	pr, err := project()
	if err != nil {
		return err
	}
	ss, err := filterSchemas(pr, args)
	if err != nil {
		return err
	}
	for _, gf := range genfuncs {
		if xx != gf.key {
			continue
		}
		return gf.gen(pr, ss)
	}
	return cor.Errorf("no generator found for %s", xx)
}

func gogen(pr *Project, ss []*dom.Schema) error {
	ppkg, err := gopkg(pr.Dir)
	if err != nil {
		return err
	}
	for _, s := range pr.Schemas {
		out := fmt.Sprintf("%s/%s_gen.go", s.Name, s.Name)
		b := gengo.NewCtx(pr.Project, s.Name, path.Join(ppkg, s.Name))
		err := gengo.WriteFile(b, filepath.Join(pr.Dir, out), s)
		if err != nil {
			return err
		}
		fmt.Println(out)
	}
	return nil
}

func pggen(pr *Project, ss []*dom.Schema) error {
	for _, s := range pr.Schemas {
		out := fmt.Sprintf("%s/%s_gen.sql", s.Name, s.Name)
		b := &gen.Ctx{Project: pr.Project, Pkg: "evt", Header: "-- generated code\n\n"}
		c := *s
		c.Models = make([]*dom.Model, 0, len(s.Models))
		for _, m := range s.Models {
			b, _ := m.Extra.Key("backup")
			if b.IsZero() {
				continue
			}
			c.Models = append(c.Models, m)
		}
		err := genpg.WriteFile(b, filepath.Join(pr.Dir, out), &c)
		if err != nil {
			return err
		}
		fmt.Println(out)
	}
	return nil
}

type Project struct {
	Dir string // project directory
	mig.History
	mig.Record
}

func project() (*Project, error) {
	path, err := mig.DiscoverProject(*dirFlag)
	if err != nil {
		return nil, err
	}
	h, err := mig.ReadHistory(path)
	if err != nil && err != mig.ErrNoHistory {
		return nil, err
	}
	return &Project{filepath.Dir(path), h, h.Curr()}, nil
}

var errNeedSchemas = cor.StrError("requires list of schema names")

func filterSchemas(pr *Project, names []string) ([]*dom.Schema, error) {
	if len(names) == 0 {
		return pr.Schemas, errNeedSchemas
	}
	ss := make([]*dom.Schema, 0, len(names))
	for _, name := range names {
		s := pr.Schema(name)
		if s == nil {
			return nil, cor.Errorf("schema %q not found", name)
		}
		ss = append(ss, s)
	}
	return ss, nil
}
