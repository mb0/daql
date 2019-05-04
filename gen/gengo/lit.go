package gengo

import (
	"github.com/mb0/daql/gen"
	"github.com/mb0/xelf/bfr"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/typ"
)

// WriteLit writes the native go literal for l to c or returns an error.
func WriteLit(c *gen.Ctx, l lit.Lit) error {
	t := l.Typ()
	opt := t.IsOpt()
	if opt && l.IsZero() {
		c.WriteString("nil")
		return nil
	}
	switch k := t.Kind; k & typ.MaskRef {
	case typ.KindNum, typ.KindBool, typ.KindInt, typ.KindReal:
		if opt {
			call := "cor.Real"
			switch k & typ.MaskElem {
			case typ.KindBool:
				call = "cor.Bool"
			case typ.KindInt:
				call = "cor.Int"
			}
			return writeCall(c, call, l)
		} else {
			c.WriteString(l.String())
		}
	case typ.KindChar, typ.KindStr:
		if opt {
			return writeCall(c, "cor.Str", l)
		} else {
			return l.WriteBfr(&bfr.Ctx{B: c.B, JSON: true})
		}
	case typ.KindRaw:
		if !opt {
			c.WriteByte('*')
		}
		return writeCall(c, "cor.Raw", l)
	case typ.KindUUID:
		if !opt {
			c.WriteByte('*')
		}
		return writeCall(c, "cor.UUID", l)
	case typ.KindTime:
		if !opt {
			c.WriteByte('*')
		}
		return writeCall(c, "cor.Time", l)
	case typ.KindSpan:
		if !opt {
			c.WriteByte('*')
		}
		return writeCall(c, "cor.Span", l)
	case typ.KindIdxr:
		c.WriteString(Import(c, "lit.List"))
		return writeIdxer(c, l)
	case typ.KindList:
		c.WriteString("[]")
		err := WriteType(c, t.Elem())
		if err != nil {
			return err
		}
		return writeIdxer(c, l)
	case typ.KindKeyr:
		c.WriteString(Import(c, "&lit.Dict"))
		return writeKeyer(c, l, func(i int, k string, e lit.Lit) error {
			c.WriteByte('{')
			err := WriteLit(c, lit.Str(k))
			if err != nil {
				return err
			}
			c.WriteString(", ")
			err = WriteLit(c, e)
			if err != nil {
				return err
			}
			return c.WriteByte('}')
		})
	case typ.KindDict:
		c.WriteString("map[string]")
		err := WriteType(c, t.Elem())
		if err != nil {
			return err
		}
		return writeKeyer(c, l, func(i int, k string, e lit.Lit) error {
			err := WriteLit(c, lit.Str(k))
			if err != nil {
				return err
			}
			c.WriteString(": ")
			return WriteLit(c, e)
		})
	case typ.KindRec, typ.KindObj:
		if opt {
			c.WriteByte('&')
		}
		t, _ := t.Deopt()
		err := WriteType(c, t)
		if err != nil {
			return err
		}
		return writeKeyer(c, l, func(i int, k string, e lit.Lit) error {
			c.WriteString(k)
			c.WriteString(": ")
			return WriteLit(c, e)
		})
	case typ.KindBits, typ.KindEnum:
		valer, ok := l.(interface{ Val() interface{} })
		if !ok {
			return cor.Errorf("expect flag or enum to implement val method got %T", l)
		}
		tref := Import(c, t.Ref)
		if opt {
			c.WriteString("&[]")
			c.WriteString(tref)
			c.WriteByte('{')
		}
		switch v := valer.Val().(type) {
		case int64:
			if t.Kind&typ.MaskRef == typ.KindEnum {
				e, ok := cor.ConstByVal(t.Consts, v)
				if !ok {
					return cor.Errorf("no constant with value %d", v)
				}
				c.WriteString(tref)
				c.WriteString(e.Cased())
			} else {
				for i, f := range cor.GetFlags(t.Consts, uint64(v)) {
					if i > 0 {
						c.WriteString(" | ")
					}
					c.WriteString(tref)
					c.WriteString(f.Cased())
				}
			}
		case string: // must be enum key
			cst, ok := cor.ConstByKey(t.Consts, v)
			if !ok {
				return cor.Errorf("no constant with key %s", v)
			}
			c.WriteString(tref)
			c.WriteString(cst.Cased())
		default:
			return cor.Errorf("unexpected constant value %T", valer.Val())
		}
		if opt {
			c.WriteString("}[0]")
		}
	}
	return nil
}

func writeCall(c *gen.Ctx, name string, l lit.Lit) error {
	c.WriteString(Import(c, name))
	c.WriteByte('(')
	err := l.WriteBfr(&bfr.Ctx{B: c.B, JSON: true})
	c.WriteByte(')')
	return err
}

func writeIdxer(c *gen.Ctx, l lit.Lit) error {
	v, ok := l.(lit.Indexer)
	if !ok {
		return cor.Errorf("expect idxer got %T", l)
	}
	c.WriteByte('{')
	n := v.Len()
	for i := 0; i < n; i++ {
		if i > 0 {
			c.WriteString(", ")
		}
		e, err := v.Idx(i)
		if err != nil {
			return err
		}
		err = WriteLit(c, e)
		if err != nil {
			return err
		}
	}
	return c.WriteByte('}')
}

func writeKeyer(c *gen.Ctx, l lit.Lit, el func(int, string, lit.Lit) error) error {
	v, ok := l.(lit.Keyer)
	if !ok {
		return cor.Errorf("expect keyer got %T", l)
	}
	c.WriteByte('{')
	keys := v.Keys()
	for i, k := range keys {
		if i > 0 {
			c.WriteString(", ")
		}
		e, err := v.Key(k)
		if err != nil {
			return err
		}
		err = el(i, k, e)
		if err != nil {
			return err
		}
	}
	return c.WriteByte('}')
}
