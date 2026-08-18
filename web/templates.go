package web

import (
	"html/template"
	"io/fs"
)

// File names are constants because each is now referenced twice: once by
// the embedded parse below, once by pages' accessors, which may re-read it
// from disk instead (see WithSourceDir).
const (
	layoutFile         = "templates/layout.html"
	homeFile           = "templates/home.html"
	loginFile          = "templates/login.html"
	signupFile         = "templates/signup.html"
	gamesNewFile       = "templates/games_new.html"
	gamesJoinFile      = "templates/games_join.html"
	gameSetupLobbyFile = "templates/game_setup_lobby.html"
	gameFile           = "templates/game.html"
)

// parsePage combines the shared layout with one page's content block into
// its own isolated *template.Template. Parsing every page together into one
// shared set wouldn't work here — each page's {{define "content"}} would
// collide with every other page's in a single shared namespace, and
// whichever was parsed last would win for all of them.
func parsePage(files ...string) *template.Template {
	all := append([]string{layoutFile}, files...)
	return template.Must(template.ParseFS(templateFS, all...))
}

var (
	homeTemplate           = parsePage(homeFile)
	loginTemplate          = parsePage(loginFile)
	signupTemplate         = parsePage(signupFile)
	gamesNewTemplate       = parsePage(gamesNewFile)
	gamesJoinTemplate      = parsePage(gamesJoinFile)
	gameSetupLobbyTemplate = parsePage(gameSetupLobbyFile)
	// gameTemplate is parsed standalone, not through parsePage — this is
	// the one page that deliberately doesn't extend layout.html's Pico.css
	// nav chrome. The game screen owns its own styling entirely, same split
	// docs/user-experience.md draws between the account/lobby flow and the
	// in-game flow.
	gameTemplate = template.Must(template.ParseFS(templateFS, gameFile))
)

// pages is a handler's handle on the page templates. Its zero value is
// production: every accessor returns the corresponding package-level
// template above, parsed once at init from the binary's embedded copy —
// exactly what the handlers used directly before this existed. With source
// set (dev mode, see WithSourceDir) the accessors ignore those and re-parse
// from the source tree on every call instead, which is what lets a template
// edit show up on the next refresh.
//
// It's a value passed to each handler rather than a package-level switch
// because a package-level one would be process-wide: a single test building
// a dev-mode mux would silently redirect every other test in the same
// binary at whatever directory it used, since all of this package's tests
// share one process (package web_test).
type pages struct {
	// source is the web/ directory as an fs.FS in dev mode, nil in
	// production.
	source fs.FS
}

func (p pages) home() (*template.Template, error)     { return p.page(homeTemplate, homeFile) }
func (p pages) login() (*template.Template, error)    { return p.page(loginTemplate, loginFile) }
func (p pages) signup() (*template.Template, error)   { return p.page(signupTemplate, signupFile) }
func (p pages) gamesNew() (*template.Template, error) { return p.page(gamesNewTemplate, gamesNewFile) }
func (p pages) gamesJoin() (*template.Template, error) {
	return p.page(gamesJoinTemplate, gamesJoinFile)
}
func (p pages) gameSetupLobby() (*template.Template, error) {
	return p.page(gameSetupLobbyTemplate, gameSetupLobbyFile)
}

// game is parsed standalone — no layout — mirroring gameTemplate's own doc
// comment above.
func (p pages) game() (*template.Template, error) { return p.parse(gameTemplate, gameFile) }

// page returns one layout-wrapped page's template.
func (p pages) page(embedded *template.Template, file string) (*template.Template, error) {
	return p.parse(embedded, layoutFile, file)
}

// parse hands back the pre-parsed embedded template unless this is a dev
// mux, in which case the files are read from disk again. Re-parsing a
// couple of small files per request is invisible next to the request
// itself, and only ever happens on a developer's machine — worth far more
// than the mtime bookkeeping caching them would need, so don't add it.
func (p pages) parse(embedded *template.Template, files ...string) (*template.Template, error) {
	if p.source == nil {
		return embedded, nil // production: parsed once at init, above
	}
	return template.ParseFS(p.source, files...)
}
