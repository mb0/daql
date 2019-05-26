package mig

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mb0/daql/dom"
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/lit"
)

const ProjectFileName = "project.daql"

var ErrNoHistory = cor.StrError("no history")
var ErrNoChanges = cor.StrError("no changes")

// DiscoverProject looks for a project file based on path and returns a cleaned path.
//
// If path points to a file it check whether the file has a project file name. If path points to a
// directory, we try to look for a project file in the current and then in all its parents.
func DiscoverProject(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		if fi.Name() == ProjectFileName {
			return path, nil
		}
		path = filepath.Dir(path)
	}
	res, err := DiscoverProject(filepath.Join(path, ProjectFileName))
	if err == nil {
		return res, nil
	}
	dir := filepath.Dir(path)
	if dir == path {
		return "", err
	}
	return DiscoverProject(dir)
}

// Record consists of a project definition and its manifest at one point in time.
// A record's path can be used to look up migration rules and scripts.
type Record struct {
	Path string // record path relative to history folder
	*dom.Project
	Manifest
}

// ReadProject reads the current project's unrecorded definition and manifest or an error.
//
// The returned record represent the current malleable project state, and may contain unrecorded
// changes and preliminary versions, not representing the eventually recorded version definition.
func ReadProject(path string) (res Record, err error) {
	res.Project, err = ResolveProject(path)
	if err != nil {
		return res, err
	}
	hdir := historyPath(res.Project, path)
	err = readFile(filepath.Join(hdir, "manifest.json"),
		func(r io.Reader) (err error) {
			res.Manifest, err = ReadManifest(r)
			return err
		})
	if err != nil {
		return res, err
	}
	res.Manifest, err = res.Update(res.Project)
	return res, err
}

// History provides project records.
type History interface {
	Path() string
	Curr() Record
	Last() Manifest
	Versions() []Version
	Manifest(v int64) (Manifest, error)
	Record(v int64) (Record, error)
	Commit(string) error
}

// ReadHistory returns the prepared project history based on a project path or an error.
//
// The history folder path defaults to '$project/hist', but can changed in the project definition.
// The folder contains a link to, or copy of the last recorded manifest and record folders each
// containing a project definition and manifest file, encoded as optionally gzipped JSON streams.
// The record folders can also contain migration rules and scripts, to migration data to that
// version. The history folder acts as staging area for migrations for unrecorded project changes.
// Record folders can have any name starting with a 'v', the daql tool uses the padded version
// number and an optional record note that only acts as memory aid. The actual record version should
// always be read from the included manifest file.
func ReadHistory(path string) (_ History, err error) {
	h := &hist{}
	h.path, err = DiscoverProject(path)
	if err != nil {
		return nil, cor.Errorf("no project file found for %q: %v", path, err)
	}
	h.curr.Project, err = ResolveProject(h.path)
	if err != nil {
		return nil, err
	}
	h.hdir = historyPath(h.curr.Project, h.path)
	dir, err := os.Open(h.hdir)
	if err != nil {
		h.curr.Manifest, err = h.curr.Manifest.Update(h.curr.Project)
		if err != nil {
			return nil, err
		}
		return h, ErrNoHistory
	}
	defer dir.Close()
	h.recs = make([]rec, 0, h.curr.First().Vers)
	fis, err := dir.Readdir(0)
	if err != nil {
		return nil, err
	}
	for _, fi := range fis {
		name := fi.Name()
		if !fi.IsDir() {
			if !strings.HasPrefix(name, "manifest.") || !isJsonStream(name) {
				continue
			}
			err = readFile(filepath.Join(h.hdir, name), func(r io.Reader) (err error) {
				h.curr.Manifest, err = ReadManifest(r)
				return err
			})
			if err != nil {
				return nil, cor.Errorf("reading %s: %v", name, err)
			}
		} else {
			if !strings.HasPrefix(name, "v") {
				continue
			}
			mpath := filepath.Join(h.hdir, name, "manifest.json")
			err = readFile(mpath, func(r io.Reader) error {
				mf, err := ReadManifest(r)
				if err == nil {
					h.recs = append(h.recs, rec{name, mf})
				}
				return err
			})
			if err != nil {
				return nil, cor.Errorf("reading %s: %v", mpath, err)
			}
		}
	}
	if len(h.recs) > 0 {
		last := h.recs[len(h.recs)-1]
		if v := h.curr.First(); v.Vers == 0 {
			h.curr.Manifest = last.Manifest
		} else if lv := last.First(); v.Vers != lv.Vers {
			return nil, cor.Errorf("inconsistent history manifest version %d != %d",
				v.Vers, lv.Vers)
		}
	}
	h.curr.Manifest, err = h.curr.Manifest.Update(h.curr.Project)
	if err != nil {
		return nil, err
	}
	return h, nil
}

func historyPath(pr *dom.Project, path string) string {
	// make sure we use a project path
	path, err := DiscoverProject(path)
	if err == nil {
		path = filepath.Dir(path)
		rel := "hist"
		if l, err := pr.Extra.Key("hist"); err == nil {
			if c, ok := l.(lit.Character); ok {
				rel = c.Char()
			}
		}
		path = filepath.Join(path, rel)
	}
	return path
}

func isJsonStream(path string) bool {
	return strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".json.gz")
}

type hist struct {
	path string
	hdir string
	curr Record
	recs []rec
}

type rec struct {
	Path string
	Manifest
}

func (h *hist) Path() string { return h.path }
func (h *hist) Curr() Record { return h.curr }
func (h *hist) Last() Manifest {
	if n := len(h.recs); n > 0 {
		return h.recs[n-1].Manifest
	}
	return nil
}

func (h *hist) Versions() []Version {
	res := make([]Version, 0, len(h.recs))
	for _, r := range h.recs {
		res = append(res, r.First())
	}
	return res
}

func (h *hist) Manifest(vers int64) (Manifest, error) {
	r, ok := h.rec(vers)
	if !ok {
		return nil, cor.Errorf("version not found")
	}
	return r.Manifest, nil
}
func (h *hist) rec(vers int64) (rec, bool) {
	for _, r := range h.recs {
		if r.First().Vers == vers {
			return r, true
		}
	}
	return rec{}, false
}

func (h *hist) Record(vers int64) (null Record, _ error) {
	r, ok := h.rec(vers)
	if !ok {
		return null, cor.Errorf("version not found")
	}
	ppath := filepath.Join(h.hdir, r.Path, "project.json")
	pr, err := readProject(ppath)
	if err != nil {
		return null, err
	}
	return Record{r.Path, pr, r.Manifest}, nil
}

func (h *hist) Commit(slug string) error {
	c := h.curr.First()
	last := h.Last()
	l := last.First()
	if c.Vers == l.Vers {
		return ErrNoChanges
	}
	err := os.MkdirAll(h.hdir, 0755)
	if err != nil {
		return cor.Errorf("create history folder %s: %v", h.hdir, err)
	}
	now := time.Now()
	rec := h.curr
	// set recording date to all changed versions
	changes := rec.Diff(last)
	for i := range rec.Manifest {
		v := &rec.Manifest[i]
		if _, ok := changes[v.Name]; ok {
			v.Date = now
		}
	}
	err = writeFile(filepath.Join(h.hdir, "manifest.json"), func(w io.Writer) error {
		_, err := rec.Manifest.WriteTo(w)
		return err
	})
	if err != nil {
		return cor.Errorf("write manifest.json: %v", err)
	}
	rec.Path = fmt.Sprintf("v%03d-%s", rec.First().Vers, now.Format("20060102"))
	if slug = cor.Keyify(slug); slug != "" {
		rec.Path = fmt.Sprintf("%s-%s", rec.Path, slug)
	}
	rdir := filepath.Join(h.hdir, rec.Path)
	err = os.MkdirAll(rdir, 0755)
	if err != nil {
		return cor.Errorf("mkdir %s: %v", rec.Path, err)
	}
	err = writeFileGz(filepath.Join(rdir, "manifest.json.gz"), func(w io.Writer) error {
		_, err := rec.Manifest.WriteTo(w)
		return err
	})
	if err != nil {
		return cor.Errorf("write manifest.json.gz: %v", err)
	}
	err = writeFileGz(filepath.Join(rdir, "project.json.gz"), func(w io.Writer) error {
		return json.NewEncoder(w).Encode(rec.Project)
	})
	if err != nil {
		return cor.Errorf("write project.json.gz: %v", err)
	}
	// TODO also move migration rule and script files, as soon as we know how to spot them.
	return nil
}

func readProject(path string) (*dom.Project, error) {
	pr := &dom.Project{}
	err := readFile(path, func(r io.Reader) error {
		return json.NewDecoder(r).Decode(pr)
	})
	if err != nil {
		return nil, err
	}
	return pr, nil
}

func readFile(path string, rf func(io.Reader) error) error {
	_, err := os.Stat(path)
	if err != nil {
		if gzipped(path) {
			return err
		}
		path = path + ".gz"
		if _, e := os.Stat(path); e != nil {
			return err
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.ReadCloser = f
	if gzipped(path) {
		r, err = gzip.NewReader(r)
		if err != nil {
			return err
		}
		defer r.Close()
	}
	return rf(r)
}
