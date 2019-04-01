package gengo

import (
	"go/format"
	"strings"

	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/gen"
	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/typ"
	"github.com/pkg/errors"
)

// Import takes a qualified name of the form 'pkg.Decl', looks up a path from context packages
// map if available, otherwise the name is used as path. If the package path is the same as the
// context package it returns the 'Decl' part. Otherwise it adds the package path to the import
// list and returns a substring starting with last package path segment: 'pkg.Decl'.
func Import(c *gen.Ctx, name string) string {
	ptr := name[0] == '*'
	if ptr {
		name = name[1:]
	}
	idx := strings.LastIndexByte(name, '.')
	var ns string
	if idx > -1 {
		ns = name[:idx]
	}
	if ns != "" && c != nil {
		if path, ok := c.Pkgs[ns]; ok {
			ns = path
		}
		if ns != c.Pkg {
			c.Imports.Add(ns)
		} else {
			name = name[idx+1:]
		}
	}
	if idx := strings.LastIndexByte(name, '/'); idx != -1 {
		name = name[idx+1:]
	}
	if ptr {
		name = "*" + name
	}
	return name
}

// WriteFile writes the elements to a go file with package and import declarations.
//
// For now only flag, enum and rec type definitions are supported
func WriteFile(c *gen.Ctx, s *dom.Schema) error {
	b := bfr.Get()
	defer bfr.Put(b)
	// swap new buffer with context buffer
	f := c.B
	c.B = b
	for _, m := range s.Models {
		c.WriteString("\n")

		err := DeclareType(c, m)
		if err != nil {
			return err
		}
	}
	// swap back
	c.B = f
	f.WriteString(c.Header)
	f.WriteString("package ")
	f.WriteString(pkgName(c.Pkg))
	f.WriteString("\n")
	if len(c.Imports.List) > 0 {
		f.WriteString("\nimport (\n")
		for _, im := range c.Imports.List {
			f.WriteString("\t\"")
			f.WriteString(im)
			f.WriteString("\"\n")
		}
		f.WriteString(")\n")
	}
	res, err := format.Source(b.Bytes())
	if err != nil {
		return err
	}
	for len(res) > 0 {
		n, err := f.Write(res)
		if err != nil {
			return err
		}
		res = res[n:]
	}
	return nil
}

// DeclareType writes a type declaration for flag, enum and rec types.
// For flag and enum types the declaration includes the constant declarations.
func DeclareType(c *gen.Ctx, m *dom.Model) (err error) {
	t := m.Type
	switch m.Kind {
	case typ.KindFlag:
		c.WriteString("type ")
		c.WriteString(m.Name)
		c.WriteString(" uint64\n\n")
		writeFlagConsts(c, t, m.Name)
	case typ.KindEnum:
		c.WriteString("type ")
		c.WriteString(m.Name)
		c.WriteString(" string\n\n")
		writeEnumConsts(c, t, m.Name)
	case typ.KindRec:
		c.WriteString("type ")
		c.WriteString(m.Name)
		c.WriteByte(' ')
		t.Kind &^= typ.FlagRef
		err = WriteType(c, t)
		c.WriteByte('\n')
	default:
		err = errors.Errorf("model kind %s cannot be declared", m.Kind)
	}
	return err
}

func pkgName(pkg string) string {
	if idx := strings.LastIndexByte(pkg, '/'); idx != -1 {
		pkg = pkg[idx+1:]
	}
	if idx := strings.IndexByte(pkg, '.'); idx != -1 {
		pkg = pkg[:idx]
	}
	return pkg
}

func refDecl(t typ.Type) string {
	if t.Info == nil {
		return ""
	}
	n := t.Ref
	if i := strings.LastIndexByte(n, '.'); i >= 0 {
		n = n[i+1:]
	}
	if len(n) > 0 {
		if c := n[0]; c < 'A' || c > 'Z' {
			n = strings.ToUpper(n[:1]) + n[1:]
		}
	}
	return n
}
func refName(t typ.Type) string {
	if t.Info == nil {
		return ""
	}
	n, fst := t.Ref, 0
	if n == "" {
		d, _ := t.Deopt()
		return "missing_" + d.Kind.String()
	}
	if i := strings.LastIndexByte(n, '.'); i >= 0 {
		fst = i + 1
	}
	if c := n[fst]; c < 'A' || c > 'Z' {
		n = n[:fst] + strings.ToUpper(n[fst:fst+1]) + n[fst+1:]
	}
	return n
}

func writeFlagConsts(c *gen.Ctx, t typ.Type, ref string) {
	mono := true
	c.WriteString("const (")
	for i, cst := range t.Consts {
		c.WriteString("\n\t")
		c.WriteString(ref)
		c.WriteString(cst.Name)
		mask := uint64(cst.Val)
		mono = mono && mask == (1<<uint64(i))
		if mono {
			if i == 0 {
				c.WriteByte(' ')
				c.WriteString(ref)
				c.WriteString(" = 1 << iota")
			}
		} else {
			c.WriteString(" = ")
			for j, cr := range cor.GetFlags(t.Consts[:i], mask) {
				if j != 0 {
					c.WriteString(" | ")
				}
				c.WriteString(ref)
				c.WriteString(cr.Name)
			}
		}
	}
	c.WriteString("\n)\n")
}

func writeEnumConsts(c *gen.Ctx, t typ.Type, ref string) {
	c.WriteString("const (")
	for _, cst := range t.Consts {
		c.WriteString("\n\t")
		c.WriteString(ref)
		c.WriteString(cst.Name)
		c.WriteByte(' ')
		c.WriteString(ref)
		c.WriteString(" = \"")
		c.WriteString(strings.ToLower(cst.Name))
		c.WriteByte('"')
	}
	c.WriteString("\n)\n")
}
