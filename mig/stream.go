package mig

import (
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lex"
	"github.com/mb0/xelf/lit"
)

// Steam represents a possibly large sequence model data object.
//
// This abstraction allows us to choose an appropriate implementation for any situation, without
// being forced to load all the data into memory at once.
type Stream interface {
	Name() string // qualified name of an object model
	Iter() (Iter, error)
}

type IOStream interface {
	Stream
	Open() (io.ReadCloser, error)
}

type Iter interface {
	Scan() (lit.Lit, error)
	Close() error
}

// FileStream is a file based stream implementation.
type FileStream struct {
	Model string
	Path  string
}

func NewFileStream(path string) FileStream {
	name := path
	if strings.HasSuffix(name, ".gz") {
		name = name[:len(name)-3]
	}
	idx := strings.LastIndexByte(name, '/')
	if idx >= 0 {
		name = name[idx+1:]
	}
	idx = strings.LastIndexByte(name, '.')
	if idx > 0 {
		name = name[:idx]
	}
	return FileStream{name, path}
}

func (s *FileStream) Name() string                 { return s.Model }
func (s *FileStream) Open() (io.ReadCloser, error) { return os.Open(s.Path) }
func (s *FileStream) Iter() (Iter, error)          { return OpenFileIter(s.Path) }

// ZipStream is a zip file based stream implementation.
type ZipStream struct {
	FileStream
	*zip.File
}

func (s *ZipStream) Open() (io.ReadCloser, error) { return s.File.Open() }
func (s *ZipStream) Iter() (Iter, error) {
	f, err := s.Open()
	if err != nil {
		return nil, err
	}
	return NewFileIter(f, gzipped(s.Path))
}

func OpenFileIter(path string) (Iter, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return NewFileIter(f, gzipped(path))
}

func NewFileIter(f io.ReadCloser, gzipped bool) (Iter, error) {
	if !gzipped {
		return &fileIter{f: f, lex: lex.New(f)}, nil
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	return &fileIter{f: f, gz: gz, lex: lex.New(gz)}, nil
}

func WriteIter(it Iter, w io.Writer) error {
	enc := json.NewEncoder(w)
	for {
		l, err := it.Scan()
		if err != nil {
			if cor.IsErr(err, io.EOF) {
				break
			}
			return err
		}
		err = enc.Encode(l)
		if err != nil {
			return err
		}
	}
	return nil
}

type fileIter struct {
	f   io.ReadCloser
	gz  *gzip.Reader
	lex *lex.Lexer
}

func (it *fileIter) Close() error {
	if it.gz != nil {
		it.gz.Close()
	}
	return it.f.Close()
}

func (it *fileIter) Scan() (lit.Lit, error) {
	tr, err := it.lex.Scan()
	if err != nil {
		return nil, err
	}
	return lit.Parse(tr)
}

func gzipped(path string) bool { return strings.HasSuffix(path, ".gz") }
