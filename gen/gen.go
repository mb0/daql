// Package gen provides code generation helpers and specific functions to generate go code.
package gen

import (
	"os"
	"sort"

	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/bfr"
)

// Ctx is the code generation context holding the buffer and additional information.
type Ctx struct {
	bfr.Ctx
	*dom.Project
	Pkg    string
	Target string
	Header string
	OpPrec int
	Pkgs   map[string]string
	Imports
}

func (c *Ctx) Prec(prec int) (restore func()) {
	org := c.OpPrec
	if org > prec {
		c.WriteByte('(')
	}
	c.OpPrec = prec
	return func() {
		if org > prec {
			c.WriteByte(')')
		}
		c.OpPrec = org
	}
}

// Imports has a list of alphabetically sorted dependencies. A dependency can be any string
// recognized by the generator. For go imports the dependency is a package path.
type Imports struct {
	List []string
}

// Add inserts path into the import list if not already present.
func (i *Imports) Add(path string) {
	idx := sort.SearchStrings(i.List, path)
	if idx < len(i.List) && i.List[idx] == path {
		return
	}
	i.List = append(i.List, "")
	copy(i.List[idx+1:], i.List[idx:])
	i.List[idx] = path
}

func DomFile(fname string, pr *dom.Project) (*dom.Schema, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if pr == nil {
		pr = &dom.Project{}
	}
	env := dom.NewEnv(dom.Env, pr)
	return dom.Execute(env, f)
}
