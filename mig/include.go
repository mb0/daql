package mig

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lit"
)

// ResolveProject returns the project schema read from the project file path or an error.
//
// This function will resolve schema includes. For now it only supports includes relative to the
// project directory are supported.
//
// We want to be able to include schema definitions from imported go packages. This however means,
// that resolution of those path is specific to the host language. We need to call the go tool to
// determine the file path to include from. Package versioning is probably enough, but we may
// need a way to specify include version and reference a project, to lookup the schema history.
// And we later need the included schema history for migration rules and scripts, so ...
//
// We should require project definitions even for library schemas, to reuse the same versioning and
// migration machinery. We then need to import the go package containing the included project file
// and declare the use of the external project in the project declaration, and can then include
// schemas from those external projects.
func ResolveProject(path string) (*dom.Project, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, cor.Errorf("open project file %s: %v", path, err)
	}
	defer f.Close()
	pr, err := dom.Read(f, nil)
	if err != nil {
		return nil, err
	}
	pdir := filepath.Dir(path)
	for _, s := range pr.Schemas {
		if inc := xstr(s.Extra, "inc", ""); inc != "" {
			ipath := filepath.FromSlash(inc)
			if !filepath.IsAbs(ipath) {
				ipath = filepath.Join(pdir, ipath)
			}
			err = includeSchema(s, ipath, s.Name)
			if err != nil {
				return nil, cor.Errorf("include %s not found: %w", inc, err)
			}
		}
	}
	return pr, nil
}

func includeSchema(s *dom.Schema, path, name string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return includeSchema(s, filepath.Join(path, fmt.Sprintf("%s.dom", name)), name)
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	pr, err := dom.Read(f, nil)
	if err != nil {
		return err
	}
	ps := pr.Schema(name)
	if ps == nil {
		return cor.Errorf("no schema %s found in %s", name, path)
	}
	*s = *ps
	return nil
}

func xlit(x *lit.Dict, key string) lit.Lit {
	l, err := x.Key(key)
	if err != nil {
		return nil
	}
	return l
}

func xstr(x *lit.Dict, key, def string) string {
	if c, ok := xlit(x, key).(lit.Character); ok {
		return c.Char()
	}
	return def
}
