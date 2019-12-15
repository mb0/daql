package qry

import (
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/exp"
	"github.com/mb0/xelf/typ"
)

type Field struct {
	Key  string
	Name string
	Type typ.Type
	El   exp.El
	Nest []*Query
}

func (f *Field) AddNested(q *Query) {
	for _, n := range f.Nest {
		if n == q {
			return
		}
	}
	f.Nest = append(f.Nest, q)
}

type Fields []*Field

func (fs Fields) Field(key string) *Field {
	for _, f := range fs {
		if f.Key == key {
			return f
		}
	}
	return nil
}

func (fs Fields) with(n *Field) (_ Fields, found bool) {
	for i, f := range fs {
		if f.Key == n.Key {
			fs[i] = n
			return fs, true
		}
	}
	return append(fs, n), false
}

func (fs Fields) without(key string) (_ Fields, found bool) {
	for i, f := range fs {
		if f.Key == key {
			return append(fs[:i], fs[i+1:]...), true
		}
	}
	return fs, false
}

type Sel struct {
	Type typ.Type
	Fields
}

func paramField(p typ.Param) *Field {
	return &Field{Key: cor.Keyed(p.Name), Name: cor.Cased(p.Name), Type: p.Type}
}

func subjFields(t typ.Type) Fields {
	if t.Kind&typ.KindRec != typ.KindRec {
		return nil
	}
	fs := make(Fields, 0, len(t.Params))
	for _, p := range t.Params {
		fs = append(fs, paramField(p))
	}
	return fs
}

func simpleExpr(el exp.El) exp.El {
	s, ok := el.(*exp.Sym)
	if ok && cor.IsKey(s.Name) {
		s := *s
		s.Name = "." + s.Name
		return &s
	}
	return el
}

func reslSel(p *exp.Prog, env exp.Env, q *Query, ds []*exp.Tag) (*Sel, error) {
	fs := subjFields(q.Subj)
	if len(ds) == 0 {
		return &Sel{Type: q.Subj, Fields: fs}, nil
	}
	var mode byte
	for i, d := range ds {
		name := d.Name
		switch name {
		case "+", "-":
			mode = name[0]
			if d.El != nil {
				return nil, cor.Errorf("unexpected selection arguments %s", d)
			}
			continue
		case "_":
			mode = '+'
			if d.El == nil {
				fs = nil
				continue
			} else if len(ds[i:]) > 1 {
				return nil, cor.Errorf("unexpected selection arguments %s", d)
			}
			renv := &ReslEnv{env, q, nil, fs}
			el, err := p.Resl(renv, simpleExpr(d.El), typ.Void)
			if err != nil && err != exp.ErrUnres {
				return nil, err
			}
			f := &Field{El: el, Type: exp.ResType(el)}
			return &Sel{Type: exp.ResType(el), Fields: Fields{f}}, nil
		case "":
			return nil, cor.Errorf("unnamed selection %s", d)
		}
		switch name[0] {
		case '-', '+':
			mode = name[0]
			name = name[1:]
		default:
		}
		key := strings.ToLower(name)
		switch mode {
		case '-': // exclude
			if d.El != nil {
				return nil, cor.Errorf("unexpected selection arguments %s", d)
			}
			fs, _ = fs.without(key)
		case '+':
			if d.El == nil { // naked selects choose a subj field by key
				p, _, err := q.Subj.ParamByKey(key)
				if err != nil {
					return nil, err
				}
				fs, _ = fs.with(paramField(*p))
			} else {
				name := cor.Cased(name)
				f := &Field{Key: key, Name: name}
				renv := &ReslEnv{env, q, f, fs}
				el, err := p.Resl(renv, d.El, typ.Void)
				if err != nil && err != exp.ErrUnres {
					return nil, err
				}
				d.El = el
				f.El = el
				f.Type = exp.ResType(d.El)
				fs, _ = fs.with(f)
			}
		}
	}
	ps := make([]typ.Param, 0, len(fs))
	for _, f := range fs {
		ps = append(ps, typ.Param{Name: f.Name, Type: f.Type})
	}
	return &Sel{Type: typ.Rec(ps), Fields: fs}, nil
}
