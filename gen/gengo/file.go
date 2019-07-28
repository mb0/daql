// Package gengo provides code generation helpers go code generation.
package gengo

import (
	"go/format"
	"io/ioutil"
	"strings"

	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/gen"
	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
	"github.com/pkg/errors"
)

func DefaultPkgs() map[string]string {
	return map[string]string{
		"cor": "github.com/mb0/xelf/cor",
		"lit": "github.com/mb0/xelf/lit",
		"typ": "github.com/mb0/xelf/typ",
		"exp": "github.com/mb0/xelf/exp",
	}
}

func NewCtx(pr *dom.Project, pkg, path string) *gen.Gen {
	return NewCtxPkgs(pr, pkg, path, DefaultPkgs())
}
func NewCtxPkgs(pr *dom.Project, pkg, path string, pkgs map[string]string) *gen.Gen {
	pkgs[pkg] = path
	return &gen.Gen{
		Project: pr, Pkg: path,
		Pkgs:   pkgs,
		Header: "// generated code\n\n",
	}
}

// Import takes a qualified name of the form 'pkg.Decl', looks up a path from context packages
// map if available, otherwise the name is used as path. If the package path is the same as the
// context package it returns the 'Decl' part. Otherwise it adds the package path to the import
// list and returns a substring starting with last package path segment: 'pkg.Decl'.
func Import(c *gen.Gen, name string) string {
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

func WriteFile(c *gen.Gen, fname string, s *dom.Schema) error {
	b := bfr.Get()
	defer bfr.Put(b)
	c.Ctx = bfr.Ctx{B: b, Tab: "\t"}
	err := RenderFile(c, s)
	if err != nil {
		return cor.Errorf("render file %s error: %v", fname, err)
	}
	err = ioutil.WriteFile(fname, b.Bytes(), 0644)
	if err != nil {
		return cor.Errorf("write file %s error: %v", fname, err)
	}
	return nil
}

// RenderFile writes the elements to a go file with package and import declarations.
//
// For now only bits, enum and rec type definitions are supported
func RenderFile(c *gen.Gen, s *dom.Schema) error {
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
		groups := groupImports(c.Imports.List, "github")
		for i, gr := range groups {
			if i != 0 {
				f.WriteByte('\n')
			}
			for _, im := range gr {
				f.WriteString("\t\"")
				f.WriteString(im)
				f.WriteString("\"\n")
			}
		}
		f.WriteString(")\n")
	}
	res, err := format.Source(b.Bytes())
	if err != nil {
		return cor.Errorf("format %s: %w", b.Bytes(), err)
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

func groupImports(list []string, pres ...string) (res [][]string) {
	other := make([]string, 0, len(list))
	rest := make([]string, 0, len(list))
Next:
	for _, im := range list {
		for _, pre := range pres {
			if strings.HasPrefix(im, pre) {
				rest = append(rest, im)
				continue Next
			}
		}
		other = append(other, im)
	}
	if len(other) > 0 {
		res = append(res, other)
	}
	if len(rest) > 0 {
		res = append(res, rest)
	}
	return res
}

// DeclareType writes a type declaration for bits, enum and rec types.
// For bits and enum types the declaration includes the constant declarations.
func DeclareType(c *gen.Gen, m *dom.Model) (err error) {
	t := m.Type
	doc, err := m.Extra.Key("doc")
	if err == nil {
		ch, ok := doc.(lit.Character)
		if ok {
			c.Prepend(ch.Char(), "// ")
		}
	}
	switch m.Type.Kind {
	case typ.KindBits:
		c.WriteString("type ")
		c.WriteString(m.Name)
		c.WriteString(" uint64\n\n")
		writeBitsConsts(c, t, m.Name)
	case typ.KindEnum:
		c.WriteString("type ")
		c.WriteString(m.Name)
		c.WriteString(" string\n\n")
		writeEnumConsts(c, t, m.Name)
	case typ.KindObj:
		c.WriteString("type ")
		c.WriteString(m.Name)
		c.WriteByte(' ')
		t.Kind &^= typ.KindCtx
		err = WriteType(c, t)
		c.WriteByte('\n')
	case typ.KindFunc:
		last := len(m.Type.Params) - 1
		c.WriteString("type ")
		if last > 0 {
			c.WriteString(m.Name)
			c.WriteString("Req ")
			err = WriteType(c, typ.Rec(m.Type.Params[:last]))
			if err != nil {
				break
			}
			c.WriteString("\n\ntype ")
		}
		c.WriteString(m.Name)
		c.WriteString("Res ")
		res := m.Type.Params[last].Type
		err = WriteType(c, typ.Rec([]typ.Param{
			{Name: "Res?", Type: res},
			{Name: "Err?", Type: typ.Str},
		}))
		if err != nil {
			break
		}
		c.WriteString("\n ")
		var tmp strings.Builder
		cc := *c
		cc.B = &tmp
		err = WriteType(&cc, res)
		if err != nil {
			break
		}
		c.Imports.Add("github.com/mb0/daql/hub")
		if last > 0 {
			c.Imports.Add("encoding/json")
			c.Fmt(`
type %[1]sFunc func(*hub.Msg, %[1]sReq) (%[2]s, error)

func (f %[1]sFunc) Serve(m *hub.Msg) interface{} {
	var req %[1]sReq
	err := json.Unmarshal(m.Raw, &req)
	if err != nil {
		return %[1]sRes{Err: err.Error()}
	}
	res, err := f(m, req)
	if err != nil {
		return %[1]sRes{Err: err.Error()}
	}
	return %[1]sRes{Res: res}
}
`, m.Name, tmp.String())
		} else {
			c.Fmt(`
type %[1]sFunc func(*hub.Msg) (%[2]s, error)

func (f %[1]sFunc) Serve(m *hub.Msg) interface{} {
	res, err := f(m)
	if err != nil {
		return %[1]sRes{Err: err.Error()}
	}
	return %[1]sRes{Res: res}
}`, m.Name, tmp.String())
		}
	default:
		err = errors.Errorf("model kind %s cannot be declared", m.Type.Kind)
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

func writeBitsConsts(c *gen.Gen, t typ.Type, ref string) {
	mono := true
	c.WriteString("const (")
	for i, cst := range t.Consts {
		c.WriteString("\n\t")
		c.WriteString(ref)
		c.WriteString(cst.Cased())
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
			for j, cr := range t.Consts[:i].Bits(mask) {
				if j != 0 {
					c.WriteString(" | ")
				}
				c.WriteString(ref)
				c.WriteString(cr.Cased())
			}
		}
	}
	c.WriteString("\n)\n")
}

func writeEnumConsts(c *gen.Gen, t typ.Type, ref string) {
	c.WriteString("const (")
	for _, cst := range t.Consts {
		c.WriteString("\n\t")
		c.WriteString(ref)
		c.WriteString(cst.Cased())
		c.WriteByte(' ')
		c.WriteString(ref)
		c.WriteString(" = \"")
		c.WriteString(cst.Key())
		c.WriteByte('"')
	}
	c.WriteString("\n)\n")
}
