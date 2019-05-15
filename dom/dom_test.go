package dom

import (
	"testing"

	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

func TestDom(t *testing.T) {
	tests := []struct {
		raw  string
		str  string
		want *Schema
	}{
		{`(schema 'test')`, `{name:'test'}`, &Schema{Common: Common{Name: "test"}}},
		{`(schema 'test' :label 'Test Schema')`, `{name:'test' label:'Test Schema'}`,
			&Schema{Common: Common{Name: "test", Extra: &lit.Dict{List: []lit.Keyed{
				{"label", lit.Str("Test Schema")},
			}}}},
		},

		{`(schema 'test' (+Dir bits +North +East +South +West))`,
			`{name:'test' models:[{name:'Dir' typ:'bits' elems:[` +
				`{name:'North' val:1} {name:'East' val:2} ` +
				`{name:'South' val:4} {name:'West' val:8}]` +
				`}]}`,
			&Schema{Common: Common{Name: "test"}, Models: []*Model{{
				Common: Common{Name: "Dir"},
				Type: typ.Type{typ.KindBits, &typ.Info{
					Ref: "test.Dir",
					Consts: typ.Constants(map[string]int64{
						"North": 1, "East": 2,
						"South": 4, "West": 8,
					}),
				}},
				Elems: []*Elem{{}, {}, {}, {}},
			}}}},
		{`(schema 'test' (+Dir enum +North +East +South +West))`,
			`{name:'test' models:[{name:'Dir' typ:'enum' elems:[` +
				`{name:'North' val:1} {name:'East' val:2} ` +
				`{name:'South' val:3} {name:'West' val:4}]` +
				`}]}`,
			&Schema{Common: Common{Name: "test"}, Models: []*Model{{
				Common: Common{Name: "Dir"},
				Type: typ.Type{typ.KindEnum, &typ.Info{
					Ref: "test.Dir",
					Consts: typ.Constants(map[string]int64{
						"North": 1, "East": 2,
						"South": 3, "West": 4,
					}),
				}},
				Elems: []*Elem{{}, {}, {}, {}},
			}}}},
		{`(schema 'test' (+Named :prop "something" +Name str))`,
			`{name:'test' models:[{name:'Named' typ:'obj' ` +
				`elems:[{name:'Name' typ:'str'}] ` +
				`prop:'something'}]}`,
			&Schema{Common: Common{Name: "test"}, Models: []*Model{{
				Common: Common{Name: "Named",
					Extra: &lit.Dict{List: []lit.Keyed{
						{"prop", lit.Str("something")},
					}},
				},
				Type: typ.Type{typ.KindObj, &typ.Info{
					Ref:    "test.Named",
					Params: []typ.Param{{Name: "Name", Type: typ.Str}},
				}},
				Elems: []*Elem{{}},
			}}}},
		{`(schema 'test' (+Named :prop :doc "something" +Name str))`,
			`{name:'test' models:[{name:'Named' typ:'obj' ` +
				`elems:[{name:'Name' typ:'str'}] ` +
				`prop:true doc:'something'}]}`,
			&Schema{Common: Common{Name: "test"}, Models: []*Model{{
				Common: Common{Name: "Named",
					Extra: &lit.Dict{List: []lit.Keyed{
						{"prop", lit.True},
						{"doc", lit.Str("something")},
					}},
				},
				Type: typ.Type{typ.KindObj, &typ.Info{
					Ref:    "test.Named",
					Params: []typ.Param{{Name: "Name", Type: typ.Str}},
				}},
				Elems: []*Elem{{}},
			}}}},
		{`(schema 'test' (+Point +X int +Y int))`,
			`{name:'test' models:[{name:'Point' typ:'obj' ` +
				`elems:[{name:'X' typ:'int'} {name:'Y' typ:'int'}]}]}`,
			&Schema{Common: Common{Name: "test"}, Models: []*Model{{
				Common: Common{Name: "Point"},
				Type: typ.Type{typ.KindObj, &typ.Info{
					Ref: "test.Point",
					Params: []typ.Param{
						{Name: "X", Type: typ.Int},
						{Name: "Y", Type: typ.Int},
					},
				}},
				Elems: []*Elem{{}, {}},
			}}}},
		{`(schema 'test' (+Named +ID uuid :pk +Name str))`,
			`{name:'test' models:[{name:'Named' typ:'obj' elems:[` +
				`{name:'ID' typ:'uuid' bits:2} {name:'Name' typ:'str'}]}]}`,
			&Schema{Common: Common{Name: "test"}, Models: []*Model{{
				Common: Common{Name: "Named"},
				Type: typ.Type{typ.KindObj, &typ.Info{
					Ref: "test.Named",
					Params: []typ.Param{
						{Name: "ID", Type: typ.UUID},
						{Name: "Name", Type: typ.Str},
					}},
				},
				Elems: []*Elem{{Bits: BitPK}, {}},
			}}}},
		{`(schema 'test' (+Foo +A str) (+Bar +B str))`, `{name:'test' models:[` +
			`{name:'Foo' typ:'obj' elems:[{name:'A' typ:'str'}]} ` +
			`{name:'Bar' typ:'obj' elems:[{name:'B' typ:'str'}]}]}`,
			&Schema{Common: Common{Name: "test"}, Models: []*Model{{
				Common: Common{Name: "Foo"},
				Type: typ.Type{typ.KindObj, &typ.Info{
					Ref: "test.Foo",
					Params: []typ.Param{
						{Name: "A", Type: typ.Str},
					},
				}},
				Elems: []*Elem{{}},
			}, {
				Common: Common{Name: "Bar"}, Type: typ.Type{typ.KindObj, &typ.Info{
					Ref: "test.Bar",
					Params: []typ.Param{
						{Name: "B", Type: typ.Str},
					},
				}},
				Elems: []*Elem{{}},
			}}}},
		{`(schema 'test' (+Foo +A str) (+Bar +B ~test.Foo))`, `{name:'test' models:[` +
			`{name:'Foo' typ:'obj' elems:[{name:'A' typ:'str'}]} ` +
			`{name:'Bar' typ:'obj' elems:[{name:'B' typ:'~test.Foo'}]}]}`,
			&Schema{Common: Common{Name: "test"}, Models: []*Model{{
				Common: Common{Name: "Foo"},
				Type: typ.Type{typ.KindObj, &typ.Info{
					Ref: "test.Foo",
					Params: []typ.Param{
						{Name: "A", Type: typ.Str},
					}},
				},
				Elems: []*Elem{{}},
			}, {
				Common: Common{Name: "Bar"},
				Type: typ.Type{typ.KindObj, &typ.Info{
					Ref: "test.Bar",
					Params: []typ.Param{
						{Name: "B", Type: typ.Obj("test.Foo")},
					}},
				},
				Elems: []*Elem{{}},
			}}}},
		{`(schema 'test' (+Group +ID str :pk) (+Entry +ID int :pk +Group str :ref '..group'))`,
			`{name:'test' models:[` +
				`{name:'Group' typ:'obj' elems:[{name:'ID' typ:'str' bits:2}]} ` +
				`{name:'Entry' typ:'obj' elems:[` +
				`{name:'ID' typ:'int' bits:2} ` +
				`{name:'Group' typ:'str' ref:'..group'}]}]}`,
			&Schema{Common: Common{Name: "test"}, Models: []*Model{{
				Common: Common{Name: "Group"},
				Type: typ.Type{typ.KindObj, &typ.Info{
					Ref: "test.Group",
					Params: []typ.Param{
						{Name: "ID", Type: typ.Str},
					}},
				},
				Elems: []*Elem{{Bits: BitPK}},
			}, {
				Common: Common{Name: "Entry"},
				Type: typ.Type{typ.KindObj, &typ.Info{
					Ref: "test.Entry",
					Params: []typ.Param{
						{Name: "ID", Type: typ.Int},
						{Name: "Group", Type: typ.Str},
					}},
				},
				Elems: []*Elem{{Bits: BitPK}, {Ref: "..group"}},
			}}}},
		{`(schema 'test' (+Spam func +Egg str + bool))`, `{name:'test' models:[` +
			`{name:'Spam' typ:'func' elems:[{name:'Egg' typ:'str'} {typ:'bool'}]}]}`,
			&Schema{Common: Common{Name: "test"}, Models: []*Model{{
				Common: Common{Name: "Spam"},
				Type: typ.Type{typ.KindFunc, &typ.Info{
					Ref: "test.Spam",
					Params: []typ.Param{
						{Name: "Egg", Type: typ.Str},
						{Type: typ.Bool},
					}},
				},
				Elems: []*Elem{{}, {}},
			}}}},
	}
	for _, test := range tests {
		env := NewEnv(Env, &Project{})
		s, err := ExecuteString(env, test.raw)
		if err != nil {
			t.Errorf("execute %s got error: %+v", test.raw, err)
			continue
		}
		got := s.String()
		want := test.want.String()
		if got != want {
			t.Errorf("string equal want\n%s\n\tgot\n%s", want, got)
		}
		if got != test.str {
			t.Errorf("string equal want %s got %s", test.str, got)
		}
		res, err := lit.ParseString(test.str)
		if err != nil {
			t.Errorf("parse %s err: %v", test.str, err)
		}
		ss := &Schema{}
		err = ss.FromDict(res.(*lit.Dict))
		if err != nil {
			t.Errorf("assign %s err: %v", res, err)
		}
		got = ss.String()
		if got != want {
			t.Errorf("parsed string equal want\n%s\n\tgot\n%s", want, got)
			continue
		}
	}
}
