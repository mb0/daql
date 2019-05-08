package dom

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"time"

	"github.com/mb0/xelf/cor"
)

// Version contains essential details for a node to derive a new version number.
//
// The name is the node's qualified name, and date is an optional recording time. Vers is a positive
// integer for known versions or zero if unknown. The hash is a lowercase hex string of an sha256
// hash of the node's qualified name and its contents. For models the default string representation
// is used as content, for schemas each model hash and for projects each schema hash.
type Version struct {
	Name string
	Vers int64
	Hash string
	Date time.Time
}

// Manifest provides version details for nodes based on records.
type Manifest interface {
	// Version returns the version for node or an error.
	Version(Node) (Version, error)
}

// NewManifest returns a new manifest with the given version records.
func NewManifest(records []Version) Manifest {
	mf := make(manifest)
	for _, v := range records {
		e := mf[v.Name]
		if e == nil {
			mf[v.Name] = &entry{old: v}
		} else if e.old.Vers < v.Vers {
			e.old = v
		}
	}
	return mf
}

type manifest map[string]*entry

type entry struct {
	old Version
	cur Version
}

func (mf manifest) Version(n Node) (res Version, err error) {
	key := n.Qualified()
	vs := mf[key]
	if vs == nil {
		res.Name = key
		res.Vers = 1
	} else if vs.cur.Vers != 0 { // we already did the work
		return vs.cur, nil
	} else if vs.old.Vers != 0 {
		res = vs.old
	} else {
		return res, cor.Errorf("internal manifest error inconsistent state")
	}
	h := sha256.New()
	io.WriteString(h, res.Name)
	switch d := n.(type) {
	case *Model:
		m := *d // local copy so we can set the version for hashing
		m.Vers = res.Vers
		io.WriteString(h, m.String())
	case *Schema:
		for _, m := range d.Models {
			mv, err := mf.Version(m)
			if err != nil {
				return res, err
			}
			io.WriteString(h, mv.Hash)
		}
	case *Project:
		for _, s := range d.Schemas {
			sv, err := mf.Version(s)
			if err != nil {
				return res, err
			}
			io.WriteString(h, sv.Hash)
		}
	default:
		return res, cor.Errorf("unexpected node type %T", n)
	}
	res.Hash = hex.EncodeToString(h.Sum(nil))
	if vs == nil {
		mf[key] = &entry{cur: res}
	} else if res.Hash != vs.old.Hash {
		res.Vers++
		vs.cur = res
	} else {
		vs.cur = vs.old
	}
	return res, nil
}
