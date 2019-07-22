package mig

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"

	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/cor"
)

// ReadVersion returns a version read from r or and error.
func ReadVersion(r io.Reader) (v Version, err error) {
	err = json.NewDecoder(r).Decode(&v)
	return v, err
}

// WriteTo writes the version to w and returns the written bytes or an error.
func (v Version) WriteTo(w io.Writer) (int64, error) {
	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(v)
	if err != nil {
		return 0, err
	}
	return b.WriteTo(w)
}

// Versioner sets and returns node version details, usually based on the last recorded manifest.
type Versioner interface {
	// Manifest returns a fresh manifest with updated versions.
	Manifest() Manifest
	// Version sets and returns the node version details or an error.
	Version(dom.Node) (Version, error)
}

// NewVersioner returns a new versioner based on the given manifest.
func NewVersioner(mf Manifest) Versioner {
	mv := make(manifestVersioner, len(mf))
	for i, v := range mf {
		key := v.Name
		if i == 0 {
			key = "_"
		}
		e := mv[key]
		if e == nil {
			mv[key] = &entry{old: v}
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

func (mv manifestVersioner) Version(n dom.Node) (res Version, err error) {
	res.Name = n.Qualified()
	key := res.Name
	if key[0] == '_' {
		key = "_"
	}
	e := mv[key]
	if e == nil {
		res.Vers = 1
	} else if e.cur.Vers != 0 { // we already did the work
		return e.cur, nil
	} else if e.old.Vers != 0 {
		res.Vers = e.old.Vers
	} else {
		return res, cor.Errorf("internal manifest error inconsistent state")
	}
	h := sha256.New()
	h.Write([]byte(res.Name))
	switch d := n.(type) {
	case *dom.Model:
		h.Write([]byte(d.String()))
	case *dom.Schema:
		for _, m := range d.Models {
			v, err := mv.Version(m)
			if err != nil {
				return res, err
			}
			h.Write([]byte(v.Hash))
		}
	case *dom.Project:
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
		res = e.old
		e.cur = res
	}
	return res, nil
}

type entry struct {
	old Version
	cur Version
}
