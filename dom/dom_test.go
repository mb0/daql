package dom

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/typ"
)

func TestDom(t *testing.T) {
	tests := []struct {
		raw  string
		want *Schema
	}{
		{`(schema 'test')`, &Schema{Name: "test"}},
		{`(schema 'test' :label 'Test Schema')`,
			&Schema{Name: "test", Display: Display{Label: "Test Schema"}}},

		{`(schema 'test' (+Dir flag +North +East +South +West))`,
			&Schema{Name: "test", Models: []*Model{
				{Name: "Dir", Kind: typ.KindFlag, Consts: []cor.Const{
					{"North", 1}, {"East", 2},
					{"South", 4}, {"West", 8},
				}},
			}}},
		{`(schema 'test' (+Dir enum +North +East +South +West))`,
			&Schema{Name: "test", Models: []*Model{
				{Name: "Dir", Kind: typ.KindEnum, Consts: []cor.Const{
					{"North", 1}, {"East", 2},
					{"South", 3}, {"West", 4},
				}},
			}}},
		{`(schema 'test' (+Named :prop "something" +Name str))`,
			&Schema{Name: "test", Models: []*Model{
				{Name: "Named", Kind: typ.KindRec, Fields: []*Field{
					{Name: "Name", Type: typ.Str},
				}, Extra: map[string]interface{}{
					"prop": "something",
				}},
			}}},
		{`(schema 'test' (+Point +X int +Y int))`,
			&Schema{Name: "test", Models: []*Model{
				{Name: "Point", Kind: typ.KindRec, Fields: []*Field{
					{Name: "X", Type: typ.Int},
					{Name: "Y", Type: typ.Int},
				}},
			}}},
		{`(schema 'test' (+Named +ID uuid :pk +Name str))`,
			&Schema{Name: "test", Models: []*Model{
				{Name: "Named", Kind: typ.KindRec, Fields: []*Field{
					{Name: "ID", Type: typ.UUID, Bits: BitPK},
					{Name: "Name", Type: typ.Str},
				}},
			}}},
		{`(schema 'test' (+Foo +A str) (+Bar +B str))`,
			&Schema{Name: "test", Models: []*Model{
				{Name: "Foo", Kind: typ.KindRec, Fields: []*Field{
					{Name: "A", Type: typ.Str},
				}},
				{Name: "Bar", Kind: typ.KindRec, Fields: []*Field{
					{Name: "B", Type: typ.Str},
				}},
			}}},
		{`(schema 'test' (+Foo +A str) (+Bar +B @Foo))`,
			&Schema{Name: "test", Models: []*Model{
				{Name: "Foo", Kind: typ.KindRec, Fields: []*Field{
					{Name: "A", Type: typ.Str},
				}},
				{Name: "Bar", Kind: typ.KindRec, Fields: []*Field{
					{Name: "B", Type: typ.Rec("test.foo")},
				}},
			}}},
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
	v, err := json.Marshal(a)
	if err != nil {
		t.Errorf("json equal error for a: %v", err)
		return false
	}
	w, err := json.Marshal(b)
	if err != nil {
		t.Errorf("json equal error for a: %v", err)
		return false
	}
	if !bytes.Equal(v, w) {
		t.Errorf("json equal want %s got %s", w, v)
		return false
	}
	return true
}
