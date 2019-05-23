package mig

import (
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"strings"

	"github.com/mb0/xelf/lex"
	"github.com/mb0/xelf/lit"
	"github.com/mb0/xelf/prx"
)

// Steam represents a possibly large sequence model data object.
//
// This abstraction allows us to choose an appropriate implementation for any situation, without
// being forced to load all the data into memory at once.
type Stream interface {
	Name() string // qualified name of an object model
	Iter() (Iter, error)
}

type Iter interface {
	Scan(obj interface{}) error
	Close() error
}

// FileStream is a file based stream implementation.
type FileStream struct {
	Model  string
	Format string
	Gzip   bool
	Path   string
}

func NewFileStream(name, path string) FileStream {
	var ext string
	gz := strings.HasSuffix(name, ".gz")
	if gz {
		name = name[:len(name)-3]
	}
	idx := strings.LastIndexByte(name, '/')
	if idx >= 0 {
		name = name[idx+1:]
	}
	idx = strings.LastIndexByte(name, '.')
	if idx > 0 {
		name, ext = name[:idx], name[idx+1:]
	}
	return FileStream{name, ext, gz, path}
}

func (s *FileStream) Name() string { return s.Model }
func (s *FileStream) Iter() (Iter, error) {
	f, err := os.Open(s.Path)
	if err != nil {
		return nil, err
	}
	return newFileIter(s, f)
}

// ZipStream is a zip file based stream implementation.
type ZipStream struct {
	FileStream
	*zip.File
}

func (s *ZipStream) Iter() (Iter, error) {
	f, err := s.File.Open()
	if err != nil {
		return nil, err
	}
	return newFileIter(&s.FileStream, f)
}

type fileIter struct {
	s   *FileStream
	f   io.ReadCloser
	gz  *gzip.Reader
	lex *lex.Lexer
}

func newFileIter(s *FileStream, f io.ReadCloser) (_ *fileIter, err error) {
	it := &fileIter{s: s, f: f}
	if s.Gzip {
		it.gz, err = gzip.NewReader(f)
		if err != nil {
			f.Close()
			return nil, err
		}
		it.lex = lex.New(it.gz)
	} else {
		it.lex = lex.New(f)
	}
	return it, nil

}

func (it *fileIter) Close() error {
	if it.gz != nil {
		it.gz.Close()
	}
	return it.f.Close()
}
func (it *fileIter) Scan(obj interface{}) error {
	tr, err := it.lex.Scan()
	if err != nil {
		return err
	}
	l, err := lit.Parse(tr)
	if err != nil {
		return err
	}
	return prx.AssignTo(l, obj)
}
