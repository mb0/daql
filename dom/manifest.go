package dom

import (
	"sort"
	"time"
)

// Version contains essential details for a node to derive a new version number.
//
// The name is the node's qualified name, and date is an optional recording time. Vers is a positive
// integer for known versions or zero if unknown. The hash is a lowercase hex string of an sha256
// hash of the node's qualified name and its contents. For models the default string representation
// is used as content, for schemas each model hash and for projects each schema hash.
type Version struct {
	Name string    `json:"name"`
	Vers int64     `json:"vers"`
	Hash string    `json:"hash"`
	Date time.Time `json:"date,omitempty"`
}

// Manifest is set of versions sorted by name, usually for all nodes of one project.
type Manifest []Version

func (mf Manifest) idx(name string) int {
	return sort.Search(len(mf), func(i int) bool { return mf[i].Name >= name })
}

func (mf Manifest) Len() int           { return len(mf) }
func (mf Manifest) Less(i, j int) bool { return mf[i].Name < mf[j].Name }
func (mf Manifest) Swap(i, j int)      { mf[i], mf[j] = mf[j], mf[i] }
func (mf Manifest) Sort() Manifest     { sort.Sort(mf); return mf }

// Get returns the version for the qualified name or false if no version was found.
func (mf Manifest) Get(name string) (Version, bool) {
	i := mf.idx(name)
	if i >= len(mf) || mf[i].Name != name {
		return Version{}, false
	}
	return mf[i], true
}

// Set inserts a version into the manifest and returns the result.
func (mf Manifest) Set(v Version) Manifest {
	i := mf.idx(v.Name)
	if i >= len(mf) {
		return append(mf, v)
	}
	if mf[i].Name != v.Name {
		mf = append(mf[:i+1], mf[i:]...)
	}
	mf[i] = v
	return mf
}

// Update sets the node versions in project and returns the updated manifest or an error.
func (mf Manifest) Update(pr *Project) (Manifest, error) {
	mv := NewVersioner(mf)
	_, err := mv.Version(pr)
	if err != nil {
		return nil, err
	}
	return mv.Manifest(), nil
}
