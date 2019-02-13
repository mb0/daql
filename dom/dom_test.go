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
					{Name: "B", Type: typ.Rec("test.Foo")},
				}},
			}}},
	}
	for _, test := range tests {
		env := NewProjectEnv(Env)
		s, err := ExecuteString(env, test.raw)
		if err != nil {
			t.Errorf("execute %s got error: %v", test.raw, err)
			continue
		}
		if !jsonEqual(s, test.want) {
			t.Errorf("for %s want %+v got %+v", test.raw, test.want, s)
			continue
		}
	}
}

func jsonEqual(a, b interface{}) bool {
	v, ev := json.Marshal(a)
	w, ew := json.Marshal(a)
	if ev != nil || ew != nil {
		return false
	}
	return bytes.Equal(v, w)
}
