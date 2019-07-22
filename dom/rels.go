package dom

import (
	"fmt"
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/typ"
)

// Rel is a bit-set describing the kind of relationship between two models referred to as a and b.
type Rel uint64

const (
	RelA1 Rel = 1 << iota // one a
	RelAN                 // many a
	RelB1                 // one b
	RelBN                 // many b

	RelEmbed   // b embedded in a
	RelRelax   // relaxed integrity checks, b might not yet be declared
	RelInter   // relation has an intermediate model, optionally with extra data
	RelReverse // reversed relation

	Rel11  = RelA1 | RelB1 // one a to one b
	Rel1N  = RelA1 | RelBN // one a to many b
	RelN1  = RelAN | RelB1 // many a to one b
	RelNN  = RelAN | RelBN // many a to many b
	RelR11 = Rel11 | RelReverse
	RelRN1 = Rel1N | RelReverse
	RelR1N = RelN1 | RelReverse
	RelRNN = RelNN | RelReverse
)

// Relation links two models a and b, optionally via a third intermediate model.
type Relation struct {
	Rel
	A, B, Via ModelRef
}

func (r Relation) String() string {
	if r.Via.Model != nil {
		return fmt.Sprintf("%s>%s>%s,", r.A, r.Via, r.B)
	}
	return fmt.Sprintf("%s>>%s, ", r.A, r.B)
}

// ModelRef is a model pointer with an optional field key.
type ModelRef struct {
	*Model
	Key string
}

func (r ModelRef) String() string {
	res := r.Model.Qualified()
	if r.Key == "" {
		return res
	}
	return fmt.Sprintf("%s.%s", res, r.Key)
}

// ModelRels contains outgoing, incoming, and intermediate relationships for a model.
type ModelRels struct {
	*Model
	Out, In, Via []Relation
}

func (r ModelRels) String() string {
	return fmt.Sprintf("{out:%v in:%v via:%v}\n\t", r.Out, r.In, r.Via)
}

// Relations maps qualified model names to a collection of all relations for that model.
type Relations map[string]*ModelRels

// Relate collects and returns all relations between the models in the given project or an error.
func Relate(pro *Project) (Relations, error) {
	res := make(Relations)
	// TODO collect relaxed reference in the first iteration
	for _, s := range pro.Schemas {
		for _, m := range s.Models {
			if !m.Type.HasParams() { // is constant
				continue
			}
			err := res.relate(pro, s, m)
			if err != nil {
				return nil, err
			}
		}
	}
	// next we check for intermediate model with at least two outgoing foreign key fields
	// TODO also wire up relaxed relations
	bs := make([]ModelRef, 0, 8)
	for _, rel := range res {
		bs = bs[:0]
		for _, r := range rel.Out {
			if r.Via.Model == nil && r.B.Key == "_" && r.Rel&RelEmbed == 0 {
				bs = append(bs, r.B)
			}
		}
		if len(bs) < 2 {
			continue
		}
		// TODO add intermediate relation for all combinations
		rel := Relation{Via: ModelRef{rel.Model, ""}, A: bs[0], B: bs[1]}
		res.add(rel)

	}
	return res, nil
}

func (res *Relations) relate(pro *Project, s *Schema, m *Model) error {
	for i, p := range m.Type.Params {
		rel := Relation{A: ModelRef{m, p.Key()}}
		e := m.Elems[i]
		if e.Ref != "" {
			if strings.HasPrefix(e.Ref, "..") {
				rel.B.Model = s.Model(cor.Keyed(e.Ref))
			} else {
				rel.B.Model = pro.Model(cor.Keyed(e.Ref))
			}
			// TODO check field type if uuid or cont|uuid or other
			rel.B.Key = "_" // signifies primary key
			if e.Bits&BitUniq != 0 {
				rel.Rel = Rel11
			} else {
				rel.Rel = RelN1
			}
		} else if embed, many := isEmbed(p.Type); embed {
			// embedded schema type
			rel.B.Model = pro.Model(p.Type.Key())
			if many {
				rel.Rel = Rel1N | RelEmbed
			} else {
				rel.Rel = Rel11 | RelEmbed
			}
		} else {
			continue
		}
		if rel.B.Model == nil {
			return cor.Errorf("model ref not found %s", e.Ref)
		}
		res.add(rel)
	}
	return nil
}

func (rs Relations) add(r Relation) {
	a := rs.upsert(r.A.Model)
	a.Out = append(a.Out, r)
	b := rs.upsert(r.B.Model)
	b.In = append(b.In, r)
	if r.Via.Model != nil {
		v := rs.upsert(r.Via.Model)
		v.Via = append(v.Via, r)
	}
}

func (rs Relations) upsert(m *Model) *ModelRels {
	key := m.Qualified()
	r := rs[key]
	if r == nil {
		r = &ModelRels{Model: m}
		rs[key] = r
	}
	return r
}

func isEmbed(t typ.Type) (yes, many bool) {
	k := t.Last().Kind
	return k&typ.KindCtx != 0 && k&typ.KindAny != 0, k != t.Kind
}
