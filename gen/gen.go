// Package gen provides generic code generation helpers.
package gen

import (
	"os"
	"sort"
	"strings"

	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/bfr"
)

// Gen is the code generation context holding the buffer and additional information.
type Gen struct {
	bfr.Ctx
	*dom.Project
	Pkg    string
	Target string
	Header string
	OpPrec int
	Pkgs   map[string]string
	Imports
}

func (c *Gen) Prec(prec int) (restore func()) {
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

// Prepend write each line in text prepended with prefix to the buffer.
// It strips the ascii whitespace bytes after the first linebreak, and tries to remove the same
// from each following line. If text starts with an empty line, that line is ignored.
func (c *Gen) Prepend(text, prefix string) {
	if text == "" {
		return
	}
	split := strings.Split(text, "\n")
	var ws int
	for i, s := range split {
		if i == 0 && s == "" && len(split) > 1 {
			continue
		}
		if i == 1 {
			for len(s) > 0 {
				switch s[0] {
				case '\t', ' ':
					ws++
					s = s[1:]
				default:
					goto Done
				}
			}
		} else {
			for j := 0; j < ws && len(s) > 0; j++ {
				switch s[0] {
				case '\t', ' ':
					s = s[1:]
				default:
					goto Done
				}
			}
		}
	Done:
		c.WriteString(prefix)
		c.WriteString(s)
		c.WriteString("\n")
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
