package mig

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/cor"
)

// Dataset consists of a project record and one or more data streams of model objects.
type Dataset struct {
	Record
	Streams []Stream
	Closer  io.Closer
}

// Close calls the closer, if configured, and should always be called.
func (d *Dataset) Close() error {
	if d.Closer != nil {
		return d.Closer.Close()
	}
	return nil
}

// ReadDataset returns a dataset with the manifest and data streams found at path or an error.
//
// Path must either point to directory or a zip file containing individual files for the manifest
// and data steams. The manifest file must be named '.manifest' and the individual data streams use
// the qualified model name with an extension for the format, that is either '.json' or '.xelf' with
// an optional '.gz', for gzipped files. The returned data streams are first read when iterated.
func ReadDataset(path string) (*Dataset, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, cor.Errorf("read data at path %q: %v", path, err)
	}
	if strings.HasPrefix(path, ".zip") {
		return zipData(f, path)
	}
	return dirData(f, path)
}

// ReadZip returns a dataset read from the given zip reader as described in ReadDataset or an error.
func ReadZip(r *zip.Reader) (_ *Dataset, err error) {
	var d Dataset
	for _, f := range r.File {
		s := ZipStream{newFileStream(f.Name, f.Name), f}
		if s.Model == ".manifest" {
			d.Manifest, err = readManifest(s.Iter())
			if err != nil {
				return nil, err
			}
			continue
		}
		d.Streams = append(d.Streams, &s)
	}
	return &d, nil
}

func dirData(f *os.File, path string) (*Dataset, error) {
	defer f.Close()
	fis, err := f.Readdir(0)
	if err != nil {
		return nil, cor.Errorf("read data dir at path %q: %v", path, err)
	}
	var d Dataset
	for _, fi := range fis {
		name := fi.Name()
		s := newFileStream(name, filepath.Join(path, name))
		if s.Model == ".manifest" {
			d.Manifest, err = readManifest(s.Iter())
			if err != nil {
				return nil, err
			}
			continue
		}
		d.Streams = append(d.Streams, &s)
	}
	return &d, nil
}

func readManifest(it Iter, err error) (dom.Manifest, error) {
	if err != nil {
		return nil, err
	}
	defer it.Close()
	mf := make(dom.Manifest, 0, 48)
	for {
		var v dom.Version
		err = it.Scan(&v)
		if err != nil {
			return nil, err
		}
		mf = append(mf, v)
	}
	return mf.Sort(), nil
}

func zipData(f *os.File, path string) (*Dataset, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, cor.Errorf("stat zip data at path %q: %v", path, err)
	}
	r, err := zip.NewReader(f, fi.Size())
	if err != nil {
		return nil, cor.Errorf("read zip data at path %q: %v", path, err)
	}
	d, err := ReadZip(r)
	if err != nil {
		f.Close()
		return nil, err
	}
	d.Closer = f
	return d, nil
}
