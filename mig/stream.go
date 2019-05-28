package mig

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lex"
	"github.com/mb0/xelf/lit"
)

// Iter is an iterator for a possibly large sequence of object literal data.
//
// This abstraction allows us to choose an appropriate implementation for any situation, without
// being forced to load all the data into memory at once.
type Iter interface {
	Scan() (lit.Lit, error)
	Close() error
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

// fileStream is a file based stream implementation.
type fileStream struct {
	Model string
	Path  string
}

func newFileStream(path string) fileStream {
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
	return fileStream{name, path}
}

func gzipped(path string) bool { return strings.HasSuffix(path, ".gz") }
