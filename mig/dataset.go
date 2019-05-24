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
)

// Dataset consists of a project version and one or more json streams of model objects.
type Dataset struct {
	Version // project version
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
// Path must either point to directory or a zip file containing individual files for the project
// version and data steams. The version file must be named 'version.json' and the individual data
// streams use the qualified model name with the '.json' extension for the format and optional a
// '.gz' extension, for gzipped files. The returned data streams are first read when iterated.
// We only allow the json format, because the files usually machine written and read and to make
// working with backups easier in other language.
func ReadDataset(path string) (*Dataset, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, cor.Errorf("read data at path %q: %v", path, err)
	}
	if zipped(path) {
		return zipData(f, path)
	}
	return dirData(f, path)
}

// WriteDataset writes a dataset to path or returns an error.  If the path ends in '.zip' a zip file
// is written, otherwise the dataset is written as individual gzipped file to the directory at path.
func WriteDataset(path string, d *Dataset) error {
	if zipped(path) {
		return writeFile(path, func(f io.Writer) error {
			w := zip.NewWriter(f)
			defer w.Close()
			return WriteZip(w, d)
		})
	}
	err := writeFile(filepath.Join(path, "version.json"), func(w io.Writer) error {
		_, err := d.Version.WriteTo(w)
		return err
	})
	if err != nil {
		return err
	}
	for _, s := range d.Streams {
		name := fmt.Sprintf("%s.json.gz", s.Name())
		err = writeFileGz(filepath.Join(path, name), func(w io.Writer) error {
			if ios, ok := s.(IOStream); ok {
				f, err := ios.Open()
				if err != nil {
					return err
				}
				_, err = io.Copy(w, f)
				f.Close()
				return err
			}
			it, err := s.Iter()
			if err != nil {
				return err
			}
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
func ReadZip(r *zip.Reader) (*Dataset, error) {
	var d Dataset
	for _, f := range r.File {
		s := ZipStream{NewFileStream(f.Name), f}
		if s.Model == "version" {
			r, err := s.Open()
			if err != nil {
				return nil, err
			}
			d.Version, err = ReadVersion(r)
			r.Close()
			if err != nil {
				return nil, err
			}
			continue
		}
		if isStream(f.Name) {
			d.Streams = append(d.Streams, &s)
		}
	}
	return &d, nil
}

// WriteZip writes a dataset to the given zip file or returns an error.
// It is the caller's responsibility to close the zip writer.
func WriteZip(z *zip.Writer, d *Dataset) error {
	w, err := z.Create("version.json")
	if err != nil {
		return err
	}
	_, err = d.Version.WriteTo(w)
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

func zipped(path string) bool { return strings.HasSuffix(path, ".zip") }
func isStream(path string) bool {
	return strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".json.gz")
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
		path := filepath.Join(path, name)
		if name == "version.json" {
			f, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			d.Version, err = ReadVersion(f)
			f.Close()
			if err != nil {
				return nil, err
			}
			continue
		}
		if isStream(name) {
			fs := NewFileStream(path)
			d.Streams = append(d.Streams, &fs)
		}
	}
	return &d, nil
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

func writeFileGz(path string, wf func(io.Writer) error) error {
	return writeFile(path, func(w io.Writer) error {
		gz := gzip.NewWriter(w)
		err := wf(gz)
		gz.Close()
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
