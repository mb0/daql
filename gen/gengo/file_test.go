package gengo

import (
	"strings"
	"testing"

	"github.com/mb0/daql/dom"
	"github.com/mb0/daql/gen"
	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/typ"
)

const barRaw = `(schema 'bar' (+Kind enum +X +Y +Z))`
const fooRaw = `(schema 'foo'
	(+Align bits
		+A
		+B
		(+C 3))
	(+Kind enum
		+A
		+B
		+C)
	(+Node1 +Name? str)
	(+Node2 +Start time)
	(+Node3 +Kind  (bits 'bar.Kind'))
	(+Node4 +Kind  @Kind)
)`

func TestWriteFile(t *testing.T) {
	env := dom.NewEnv(dom.Env, &dom.Project{})
	_, err := dom.ExecuteString(env, barRaw)
	if err != nil {
		t.Fatalf("schema bar error %v", err)
	}
	s, err := dom.ExecuteString(env, fooRaw)
	if err != nil {
		t.Fatalf("schema foo error %v", err)
	}
	tests := []struct {
		model string
		want  string
	}{
		{"", "package foo\n"},
		{"align",
			"package foo\n\ntype Align uint64\n\n" +
				"const (\n" +
				"\tAlignA Align = 1 << iota\n" +
				"\tAlignB\n" +
				"\tAlignC = AlignA | AlignB\n" +
				")\n",
		},
		{"kind",
			"package foo\n\ntype Kind string\n\n" +
				"const (\n" +
				"\tKindA Kind = \"a\"\n" +
				"\tKindB Kind = \"b\"\n" +
				"\tKindC Kind = \"c\"\n" +
				")\n",
		},
		{"node1", "package foo\n\ntype Node1 struct {\n" +
			"\tName string `json:\"name,omitempty\"`\n" + "}\n",
		},
		{"node2", "package foo\n\nimport (\n\t\"time\"\n)\n\ntype Node2 struct {\n" +
			"\tStart time.Time `json:\"start\"`\n" + "}\n",
		},
		{"node3", "package foo\n\nimport (\n\t\"path/to/bar\"\n)\n\ntype Node3 struct {\n" +
			"\tKind bar.Kind `json:\"kind\"`\n" + "}\n",
		},
		{"node4", "package foo\n\ntype Node4 struct {\n" +
			"\tKind Kind `json:\"kind\"`\n" + "}\n",
		},
	}
	pkgs := map[string]string{
		"cor": "github.com/mb0/xelf/cor",
		"foo": "path/to/foo",
		"bar": "path/to/bar",
	}
	for _, test := range tests {
		var b strings.Builder
		c := &gen.Gen{Ctx: bfr.Ctx{B: &b}, Pkg: "path/to/foo", Pkgs: pkgs}
		ss := &dom.Schema{Common: dom.Common{Name: s.Name}}
		if m := s.Model(test.model); m != nil {
			ss.Models = []*dom.Model{m}
		}
		err := RenderFile(c, ss)
		if err != nil {
			t.Errorf("write %s error: %v", test.model, err)
			continue
		}
		if got := b.String(); got != test.want {
			t.Errorf("for %s want %s got %s", test.model, test.want, got)
		}
	}
}

func rec(ref string, fs []typ.Param) typ.Type {
	res := typ.Obj(ref)
	res.Params = fs
	return res
}
