package mig

import (
	"bytes"
	"encoding/json"
	"io"
	"sort"

	"github.com/mb0/daql/dom"
)

// Manifest is set of versions sorted by name, usually for all nodes of one project.
// Manifest usually contain exactly one project version as first element, due to the project name
// qualifier prefix.
type Manifest []Version

// ReadManifest returns a manifest read from r or an error.
// Manifests are read as JSON stream of version objects.
func ReadManifest(r io.Reader) (mf Manifest, err error) {
	dec := json.NewDecoder(r)
	for {
		var v Version
		err = dec.Decode(&v)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		mf = append(mf, v)
	}
	return mf.Sort(), nil
}

func (mf Manifest) First() (v Version) {
	if len(mf) > 0 {
		return mf[0]
	}
	return v
}

// WriteTo writes the manifest to w and returns the written bytes or an error.
// Manifests are written as JSON stream of version objects.
func (mf Manifest) WriteTo(w io.Writer) (nn int64, err error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, v := range mf {
		err = enc.Encode(v)
		if err != nil {
			return nn, err
		}
		n, err := buf.WriteTo(w)
		nn += n
		if err != nil {
			return nn, err
		}
	}
	return nn, nil
}

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

func (mf Manifest) Len() int           { return len(mf) }
func (mf Manifest) Less(i, j int) bool { return mf[i].Name < mf[j].Name }
func (mf Manifest) Swap(i, j int)      { mf[i], mf[j] = mf[j], mf[i] }
func (mf Manifest) Sort() Manifest     { sort.Sort(mf); return mf }

func (mf Manifest) idx(name string) int {
	return sort.Search(len(mf), func(i int) bool { return mf[i].Name >= name })
}
