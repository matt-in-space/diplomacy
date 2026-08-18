package web_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/web"
)

// writeDevFile writes one file into a source tree (creating parent
// directories as needed), keyed by a path relative to the tree root, e.g.
// "templates/home.html" or "static/game.css" — the same shape as the real
// repository's web/ directory that web.WithSourceDir points at.
func writeDevFile(t *testing.T, dir, path, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", full, err)
	}
}

// A trivial layout is enough for every test below: none of them render
// anything the real layout.html adds (nav, flash), so there's no reason to
// duplicate it here.
const testLayout = `{{define "layout"}}{{template "content" .}}{{end}}`

func TestDevModeRendersTemplateEditsWithoutRestart(t *testing.T) {
	dir := t.TempDir()
	writeDevFile(t, dir, "templates/layout.html", testLayout)
	writeDevFile(t, dir, "templates/home.html", `{{define "content"}}MARKER-A{{end}}`)

	lobbyService, gameplayService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService, gameplayService, web.WithSourceDir(dir))

	get := func() string {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		return rec.Body.String()
	}

	if body := get(); !strings.Contains(body, "MARKER-A") {
		t.Fatalf("body = %q, want it to contain %q", body, "MARKER-A")
	}

	// The edit a developer would make mid-session — no server restart, same
	// mux, same process.
	writeDevFile(t, dir, "templates/home.html", `{{define "content"}}MARKER-B{{end}}`)

	if body := get(); !strings.Contains(body, "MARKER-B") {
		t.Fatalf("body after edit = %q, want it to contain %q (not the stale %q)", body, "MARKER-B", "MARKER-A")
	}
}

// TestDevModeDoesNotAffectOtherMuxes pins the reason pages is per-mux state
// rather than a package-level switch (see pages' doc comment in
// templates.go): a dev mux built in one test must never leak into a
// default (embedded) mux built anywhere else in this shared test binary.
func TestDevModeDoesNotAffectOtherMuxes(t *testing.T) {
	dir := t.TempDir()
	writeDevFile(t, dir, "templates/layout.html", testLayout)
	writeDevFile(t, dir, "templates/home.html", `{{define "content"}}MARKER-DEV{{end}}`)

	lobbyService, gameplayService := newTestLobbyService(t)
	_ = web.NewMux(newTestAuthService(t), lobbyService, gameplayService, web.WithSourceDir(dir))

	defaultLobbyService, defaultGameplayService := newTestLobbyService(t)
	defaultMux := web.NewMux(newTestAuthService(t), defaultLobbyService, defaultGameplayService)

	rec := httptest.NewRecorder()
	defaultMux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	body := rec.Body.String()
	if !strings.Contains(body, "Hello, Diplomacy") {
		t.Fatalf("default mux body = %q, want the real embedded home page", body)
	}
	if strings.Contains(body, "MARKER-DEV") {
		t.Fatalf("default mux body = %q, leaked content from an unrelated dev mux", body)
	}
}

func TestDevModeServesStaticFromDisk(t *testing.T) {
	dir := t.TempDir()
	writeDevFile(t, dir, "static/dev.css", "body{color:red}")

	lobbyService, gameplayService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService, gameplayService, web.WithSourceDir(dir))

	get := func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/dev.css", nil))
		return rec
	}

	rec := get()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); body != "body{color:red}" {
		t.Fatalf("body = %q, want %q", body, "body{color:red}")
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q, want %q", cc, "no-store")
	}

	writeDevFile(t, dir, "static/dev.css", "body{color:blue}")

	if body := get().Body.String(); body != "body{color:blue}" {
		t.Fatalf("body after edit = %q, want %q", body, "body{color:blue}")
	}
}

// TestDevModeDoesNotServeUnderscoredPaths is the dev-mode twin of
// TestStaticDoesNotServeJSTests in home_test.go: go:embed silently excludes
// any '_'/'.'-prefixed path when it walks a directory, which is what keeps
// web/static/js/_tests/ off the /static/ route in production. os.DirFS
// doesn't apply that rule on its own, so dev mode's embedRules wrapper
// (devfs.go) has to reproduce it — otherwise the exclusion becomes a
// property of how the server was started, not of the route itself.
func TestDevModeDoesNotServeUnderscoredPaths(t *testing.T) {
	dir := t.TempDir()
	writeDevFile(t, dir, "static/js/main.mjs", "export {}")
	writeDevFile(t, dir, "static/js/_tests/x.mjs", "test")
	writeDevFile(t, dir, "static/.hidden", "secret")

	lobbyService, gameplayService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService, gameplayService, web.WithSourceDir(dir))

	for _, path := range []string{"/static/js/_tests/x.mjs", "/static/.hidden"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want %d", path, rec.Code, http.StatusNotFound)
		}
	}

	// The other half of the exclusion: a directory listing has to hide the
	// same names Open refuses, or the route would advertise a file it then
	// 404s on — this is the only coverage of embedRules' ReadDir filtering.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/js/", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "main.mjs") {
		t.Fatalf("listing = %q, want it to mention %q", body, "main.mjs")
	}
	if strings.Contains(body, "_tests") {
		t.Fatalf("listing = %q, want it to omit %q", body, "_tests")
	}
}

// TestDevModeSurvivesTemplateParseError pins that a mid-edit typo produces
// an ordinary 500 with the parse error in the body — not a panic that would
// take the dev server down, which is what production's template.Must at
// package init would do for the same mistake.
func TestDevModeSurvivesTemplateParseError(t *testing.T) {
	dir := t.TempDir()
	writeDevFile(t, dir, "templates/layout.html", testLayout)
	writeDevFile(t, dir, "templates/home.html", `{{define "content"}}{{end}}{{end}}`) // stray {{end}}

	lobbyService, gameplayService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService, gameplayService, web.WithSourceDir(dir))

	get := func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		return rec
	}

	rec := get()
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if body := rec.Body.String(); !strings.Contains(strings.ToLower(body), "template") {
		t.Fatalf("body = %q, want it to mention the template parse error", body)
	}

	writeDevFile(t, dir, "templates/home.html", `{{define "content"}}MARKER-FIXED{{end}}`)

	rec = get()
	if rec.Code != http.StatusOK {
		t.Fatalf("status after fix = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, "MARKER-FIXED") {
		t.Fatalf("body after fix = %q, want it to contain %q", body, "MARKER-FIXED")
	}
}
