package dom

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/mb0/xelf/cor"
)

// Versioner sets and returns node version details, usually based on the last recorded manifest.
type Versioner interface {
	// Manifest returns a fresh manifest with updated versions.
	Manifest() Manifest
	// Version sets and returns the node version details or an error.
	Version(Node) (Version, error)
}

// NewVersioner returns a new versioner based on the given manifest.
func NewVersioner(mf Manifest) Versioner {
	mv := make(manifestVersioner, len(mf))
	for _, v := range mf {
		e := mv[v.Name]
		if e == nil {
			mv[v.Name] = &entry{old: v}
		} else if e.old.Vers < v.Vers {
			e.old = v
		}
	}
	return mv
}

type manifestVersioner map[string]*entry

func (mv manifestVersioner) Manifest() Manifest {
	mf := make(Manifest, 0, len(mv))
	for _, e := range mv {
		if e.cur.Vers != 0 {
			mf = append(mf, e.cur)
		} else {
			mf = append(mf, e.old)
		}
	}
	return mf.Sort()

}

func (mv manifestVersioner) Version(n Node) (res Version, err error) {
	key := n.Qualified()
	e := mv[key]
	if e == nil {
		res.Name = key
		res.Vers = 1
	} else if e.cur.Vers != 0 { // we already did the work
		return e.cur, nil
	} else if e.old.Vers != 0 {
		res = e.old
	} else {
		return res, cor.Errorf("internal manifest error inconsistent state")
	}
	h := sha256.New()
	h.Write([]byte(res.Name))
	var c *Common
	switch d := n.(type) {
	case *Model:
		c = &d.Common
		m := *d // local copy so we can set the version for hashing
		m.Vers = res.Vers
		h.Write([]byte(m.String()))
	case *Schema:
		c = &d.Common
		for _, m := range d.Models {
			v, err := mv.Version(m)
			if err != nil {
				return res, err
			}
			h.Write([]byte(v.Hash))
		}
	case *Project:
		c = &d.Common
		for _, s := range d.Schemas {
			v, err := mv.Version(s)
			if err != nil {
				return res, err
			}
			h.Write([]byte(v.Hash))
		}
	default:
		return res, cor.Errorf("unexpected node type %T", n)
	}
	res.Hash = hex.EncodeToString(h.Sum(nil))
	if e == nil {
		mv[key] = &entry{cur: res}
	} else if res.Hash != e.old.Hash {
		res.Vers++
		e.cur = res
	} else {
		e.cur = e.old
	}
	c.Vers = res.Vers
	return res, nil
}

type entry struct {
	old Version
	cur Version
}
