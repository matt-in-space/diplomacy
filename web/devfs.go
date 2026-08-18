package web

import (
	"io"
	"io/fs"
	"strings"
)

// embedRules mimics the one rule go:embed applies to an embedded directory
// that a plain os.DirFS doesn't: files and directories whose names begin
// with '_' or '.' aren't there. web/static/js/_tests/ relies on this to stay
// out of the binary and off the /static/ route (see
// TestStaticDoesNotServeJSTests in web/home_test.go); dev mode reading the
// same tree from disk has to agree, or the exclusion silently becomes a
// property of how the server was started — and along the way, editor junk
// like .swp/.DS_Store files becomes servable too.
//
// Only static needs this. go:embed's exclusion applies when it *walks* a
// directory (the `//go:embed static` form here); a glob that matches files
// directly (`//go:embed templates/*.html`) embeds underscore-prefixed files
// same as any other, so templates behave identically with or without this
// wrapper.
type embedRules struct{ fsys fs.FS }

func (e embedRules) Open(name string) (fs.File, error) {
	if excluded(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	f, err := e.fsys.Open(name)
	if err != nil {
		return nil, err
	}
	// Directory listings have to hide the same names Open refuses, or
	// http.FileServerFS would advertise files it then 404s on request.
	if dir, ok := f.(fs.ReadDirFile); ok {
		return excludingDir{dir}, nil
	}
	return f, nil
}

// excluded reports whether any path element is one go:embed would have
// skipped. "." is the root fs.FS callers open to list a directory (e.g.
// http.FileServerFS opens it for every "/static/" request) — not a hidden
// name — so it must not match, or every static request 404s.
func excluded(name string) bool {
	for _, elem := range strings.Split(name, "/") {
		if elem == "." {
			continue
		}
		if strings.HasPrefix(elem, "_") || strings.HasPrefix(elem, ".") {
			return true
		}
	}
	return false
}

type excludingDir struct{ fs.ReadDirFile }

func (d excludingDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if n <= 0 {
		entries, err := d.ReadDirFile.ReadDir(n)
		return visible(entries), err
	}
	// n > 0 means "at most n entries, io.EOF once exhausted" — filtering a
	// single batch could come up short while entries remain, so keep
	// reading until n survive the filter or the directory runs out.
	kept := make([]fs.DirEntry, 0, n)
	for len(kept) < n {
		entries, err := d.ReadDirFile.ReadDir(n - len(kept))
		kept = append(kept, visible(entries)...)
		if err != nil {
			return kept, err
		}
		if len(entries) == 0 {
			return kept, io.EOF
		}
	}
	return kept, nil
}

func visible(entries []fs.DirEntry) []fs.DirEntry {
	kept := entries[:0:0]
	for _, entry := range entries {
		if !excluded(entry.Name()) {
			kept = append(kept, entry)
		}
	}
	return kept
}
