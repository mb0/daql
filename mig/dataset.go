package mig

import (
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/prx"
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
// and data steams. The manifest file must be named 'manifest' and the individual data streams use
// the qualified model name with an extension for the format, that is either '.json' or '.xelf' with
// an optional '.gz', for gzipped files. The returned data streams are first read when iterated.
func ReadDataset(path string) (*Dataset, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, cor.Errorf("read data at path %q: %v", path, err)
	}
	if strings.HasSuffix(path, ".zip") {
		return zipData(f, path)
	}
	return dirData(f, path)
}

// WriteDataset writes a dataset to path or returns an error.  If the path ends in '.zip' a zip file
// is written, otherwise the dataset is written as individual gzipped file to the directory at path.
func WriteDataset(path string, d *Dataset) error {
	if strings.HasSuffix(path, ".zip") {
		return writeFile(path, func(f io.Writer) error {
			w := zip.NewWriter(f)
			defer w.Close()
			return WriteZip(w, d)
		})
	}
	gz := new(gzip.Writer)
	defer gz.Close()
	err := writeFileGz(filepath.Join(path, "manifest.json.gz"), gz, func(w io.Writer) error {
		return WriteManifest(d.Manifest, w)
	})
	if err != nil {
		return err
	}
	for _, s := range d.Streams {
		it, err := s.Iter()
		if err != nil {
			return err
		}
		name := fmt.Sprintf("%s.json.gz", s.Name())
		err = writeFileGz(filepath.Join(path, name), gz, func(w io.Writer) error {
			return WriteIter(it, w)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadZip returns a dataset read from the given zip reader as described in ReadDataset or an error.
// It is the caller's responsibility to close a zip read closer or any underlying reader.
func ReadZip(r *zip.Reader) (_ *Dataset, err error) {
	var d Dataset
	for _, f := range r.File {
		s := ZipStream{NewFileStream(f.Name), f}
		if s.Model == "manifest" {
			d.Manifest, err = ReadManifestStream(&s)
			if err != nil {
				return nil, err
			}
			continue
		}
		d.Streams = append(d.Streams, &s)
	}
	return &d, nil
}

// WriteZip writes a dataset to the given zip file or returns an error.
// It is the caller's responsibility to close the zip writer.
func WriteZip(z *zip.Writer, d *Dataset) error {
	w, err := z.Create("manifest.json")
	if err != nil {
		return err
	}
	err = WriteManifest(d.Manifest, w)
	if err != nil {
		return err
	}
	for _, s := range d.Streams {
		it, err := s.Iter()
		if err != nil {
			return err
		}
		w, err = z.Create(fmt.Sprintf("%s.json", s.Name()))
		if err != nil {
			return err
		}
		err = WriteIter(it, w)
		if err != nil {
			return err
		}
	}
	z.Flush()
	return nil
}

func dirData(f *os.File, path string) (*Dataset, error) {
	defer f.Close()
	fis, err := f.Readdir(0)
	if err != nil {
		return nil, cor.Errorf("read data dir at path %q: %v", path, err)
	}
	var d Dataset
	for _, fi := range fis {
		s := NewFileStream(filepath.Join(path, fi.Name()))
		if s.Model == "manifest" {
			d.Manifest, err = ReadManifestStream(&s)
			if err != nil {
				return nil, err
			}
			continue
		}
		d.Streams = append(d.Streams, &s)
	}
	return &d, nil
}

func ReadManifestStream(s Stream) (Manifest, error) {
	it, err := s.Iter()
	if err != nil {
		return nil, err
	}
	defer it.Close()
	mf := make(Manifest, 0, 48)
	for {
		l, err := it.Scan()
		if err != nil {
			return nil, err
		}
		var v Version
		prx.AssignTo(l, &v)
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

func writeFileGz(path string, gz *gzip.Writer, wf func(io.Writer) error) error {
	return writeFile(path, func(w io.Writer) error {
		gz.Reset(w)
		err := wf(gz)
		gz.Flush()
		return err
	})
}

func writeFile(path string, wf func(io.Writer) error) error {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	err = wf(f)
	f.Close()
	return err
}
