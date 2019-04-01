package dom

import (
	"bytes"
	"testing"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

func TestDom(t *testing.T) {
	tests := []struct {
		raw  string
		want *Schema
	}{
		{`(schema 'test')`, &Schema{Node: Node{Name: "test"}}},
		{`(schema 'test' :label 'Test Schema')`, &Schema{
			Node: Node{Name: "test"},
			Extra: &lit.Dict{List: []lit.Keyed{
				{"label", lit.Str("Test Schema")},
			},
			}}},

		{`(schema 'test' (+Dir flag +North +East +South +West))`,
			&Schema{Node: Node{Name: "test"}, Models: []*Model{{
				Node: Node{Name: "Dir"},
				Type: typ.Type{typ.KindFlag, &typ.Info{
					Ref: "test.Dir",
					Consts: []cor.Const{
						{"North", 1}, {"East", 2},
						{"South", 4}, {"West", 8},
					},
				}},
				Elems: []*Elem{{}, {}, {}, {}},
			}}}},
		{`(schema 'test' (+Dir enum +North +East +South +West))`,
			&Schema{Node: Node{Name: "test"}, Models: []*Model{{
				Node: Node{Name: "Dir"},
				Type: typ.Type{typ.KindEnum, &typ.Info{
					Ref: "test.Dir",
					Consts: []cor.Const{
						{"North", 1}, {"East", 2},
						{"South", 3}, {"West", 4},
					},
				}},
				Elems: []*Elem{{}, {}, {}, {}},
			}}}},
		{`(schema 'test' (+Named :prop "something" +Name str))`,
			&Schema{Node: Node{Name: "test"}, Models: []*Model{{
				Node: Node{Name: "Named"},
				Extra: &lit.Dict{List: []lit.Keyed{
					{"prop", lit.Str("something")},
				}},
				Type: typ.Type{typ.KindRec, &typ.Info{
					Ref:    "test.Named",
					Params: []typ.Param{{Name: "Name", Type: typ.Str}},
				}},
				Elems: []*Elem{{}},
			}}}},
		{`(schema 'test' (+Point +X int +Y int))`,
			&Schema{Node: Node{Name: "test"}, Models: []*Model{{
				Node: Node{Name: "Point"},
				Type: typ.Type{typ.KindRec, &typ.Info{
					Ref: "test.Point",
					Params: []typ.Param{
						{Name: "X", Type: typ.Int},
						{Name: "Y", Type: typ.Int},
					},
				}},
				Elems: []*Elem{{}, {}},
			}}}},
		{`(schema 'test' (+Named +ID uuid :pk +Name str))`,
			&Schema{Node: Node{Name: "test"}, Models: []*Model{{
				Node: Node{Name: "Named"},
				Type: typ.Type{typ.KindRec, &typ.Info{
					Ref: "test.Named",
					Params: []typ.Param{
						{Name: "ID", Type: typ.UUID},
						{Name: "Name", Type: typ.Str},
					}},
				},
				Elems: []*Elem{{Bits: BitPK}, {}},
			}}}},
		{`(schema 'test' (+Foo +A str) (+Bar +B str))`,
			&Schema{Node: Node{Name: "test"}, Models: []*Model{{
				Node: Node{Name: "Foo"},
				Type: typ.Type{typ.KindRec, &typ.Info{
					Ref: "test.Foo",
					Params: []typ.Param{
						{Name: "A", Type: typ.Str},
					},
				}},
				Elems: []*Elem{{}},
			}, {
				Node: Node{Name: "Bar"}, Type: typ.Type{typ.KindRec, &typ.Info{
					Ref: "test.Bar",
					Params: []typ.Param{
						{Name: "B", Type: typ.Str},
					},
				}},
				Elems: []*Elem{{}},
			}}}},
		{`(schema 'test' (+Foo +A str) (+Bar +B @Foo))`,
			&Schema{Node: Node{Name: "test"}, Models: []*Model{{
				Node: Node{Name: "Foo"},
				Type: typ.Type{typ.KindRec, &typ.Info{
					Ref: "test.Foo",
					Params: []typ.Param{
						{Name: "A", Type: typ.Str},
					}},
				},
				Elems: []*Elem{{}},
			}, {
				Node: Node{Name: "Bar"},
				Type: typ.Type{typ.KindRec, &typ.Info{
					Ref: "test.Bar",
					Params: []typ.Param{
						{Name: "B", Type: typ.Rec("test.Foo")},
					}},
				},
				Elems: []*Elem{{}},
			}}}},
	}
	for _, test := range tests {
		env := NewEnv(Env, &Project{})
		s, err := ExecuteString(env, test.raw)
		if err != nil {
			t.Errorf("execute %s got error: %+v", test.raw, err)
			continue
		}
		if !jsonEqual(t, s, test.want) {
			continue
		}
	}
}

func jsonEqual(t *testing.T, a, b interface{}) bool {
	t.Helper()
	x, err := lit.Proxy(a)
	if err != nil {
		t.Errorf("proxy %T error: %v", a, err)
		return false
	}
	y, err := lit.Proxy(b)
	if err != nil {
		t.Errorf("proxy %T error: %v", b, err)
		return false
	}
	v, err := x.MarshalJSON()
	if err != nil {
		t.Errorf("marshal json %T error: %v", a, err)
		return false
	}
	w, err := y.MarshalJSON()
	if err != nil {
		t.Errorf("marshal json %T error: %v", b, err)
		return false
	}

	if !bytes.Equal(v, w) {
		t.Errorf("json equal want %s got %s", w, v)
		return false
	}
	return true
}
