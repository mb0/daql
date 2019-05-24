package mig

import (
	"encoding/json"
	"io"
	"sort"

	"github.com/mb0/daql/dom"
)

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
func (mf Manifest) Update(pr *dom.Project) (Manifest, error) {
	mv := NewVersioner(mf)
	_, err := mv.Version(pr)
	if err != nil {
		return nil, err
	}
	return mv.Manifest(), nil
}

// ReadManifest returns a manifest read from r or an error.
// Manifests are read as JSON stream of version objects.
func ReadManifest(r io.Reader) (mf Manifest, err error) {
	dec := json.NewDecoder(r)
	for {
		var v Version
		err = dec.Decode(&v)
		if err != nil {
			return nil, err
		}
		mf = append(mf, v)
	}
	return mf, nil
}

// WriteManifest writes the manifest to w or returns an error.
// Manifests are written as JSON stream of version objects.
func WriteManifest(mf Manifest, w io.Writer) error {
	enc := json.NewEncoder(w)
	for _, v := range mf {
		err := enc.Encode(v)
		if err != nil {
			return err
		}
	}
	return nil
}
