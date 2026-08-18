package web

import (
	"io/fs"
	"net/http"
	"os"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/application/lobby"
)

// Option configures optional NewMux behavior. Options are variadic rather
// than a positional parameter because every existing caller wants the
// defaults, and there are dozens of them (mostly tests) — a positional
// parameter would mean a mechanical edit to each for no gain.
type Option func(*config)

type config struct {
	// sourceDir is the repository's web/ directory when templates and
	// static assets should be read from disk per request, or "" (the
	// default) to serve the copies embedded in the binary.
	sourceDir string
}

// WithSourceDir serves templates and static assets from dir — the
// repository's web/ directory — instead of from the binary's embedded
// copy, re-reading each file as it's needed so an edit shows up on the next
// refresh with no rebuild or restart. Development only: it ties a running
// server to a source tree whose presence it can't itself guarantee (the
// caller is expected to check that first — see cmd/server's devSourceDir)
// and gives up the single-self-contained-binary property the embedded
// default exists for.
func WithSourceDir(dir string) Option {
	return func(c *config) { c.sourceDir = dir }
}

// NewMux returns the full application handler: routes plus the global
// middleware chain. It's an http.Handler, not a *http.ServeMux, because
// once wrapped in middleware it genuinely isn't a bare mux anymore — and
// nothing (callers, tests) needs it to be, since ServeHTTP is all
// http.ListenAndServe or httptest.NewRecorder-based tests ever call.
func NewMux(authService *auth.Service, lobbyService *lobby.Service, gameplayService *gameplay.GameplayService, opts ...Option) http.Handler {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	mux := http.NewServeMux()

	tmpl, staticHandler := sources(cfg.sourceDir)

	mux.Handle("GET /", handleHome(tmpl, lobbyService))
	mux.Handle("GET /static/", http.StripPrefix("/static/", staticHandler))

	mux.Handle("GET /signup", redirectIfAuthenticated(handleSignupForm(tmpl)))
	mux.HandleFunc("POST /signup", handleSignupSubmit(authService))
	mux.Handle("GET /login", redirectIfAuthenticated(handleLoginForm(tmpl)))
	mux.HandleFunc("POST /login", handleLoginSubmit(authService))
	mux.HandleFunc("POST /logout", handleLogout(authService))

	mux.Handle("GET /games/new", requireAuthentication(handleNewGameForm(tmpl)))
	mux.Handle("POST /games", requireAuthentication(handleCreateGame(lobbyService)))
	mux.Handle("GET /games/join", requireAuthentication(handleJoinGameForm(tmpl)))
	mux.Handle("POST /games/join", requireAuthentication(handleJoinGameSubmit(lobbyService)))
	mux.Handle("GET /games/{id}/lobby", requireAuthentication(handleGameSetupLobby(tmpl, lobbyService, authService)))
	mux.Handle("POST /games/{id}/start", requireAuthentication(handleStartGame(lobbyService)))
	mux.Handle("GET /games/{id}", requireAuthentication(handleGame(tmpl, lobbyService)))
	mux.Handle("GET /games/{id}/state", requireAuthentication(handleGameState(lobbyService, gameplayService)))

	// Global: more than just auth routes will eventually want to know who's
	// asking (nav, later pages). Never blocks a request — see
	// withCurrentPlayer's doc comment.
	return withCurrentPlayer(authService)(mux)
}

// sources resolves where page templates and static assets are read from:
// the embedded copies by default, or sourceDir's tree (dev mode) when set.
func sources(sourceDir string) (pages, http.Handler) {
	if sourceDir == "" {
		staticRoot, err := fs.Sub(staticFS, "static")
		if err != nil {
			panic(err) // fs.Sub only rejects a malformed subdirectory name; "static" isn't one
		}
		return pages{}, http.FileServerFS(staticRoot)
	}

	root := os.DirFS(sourceDir)
	staticRoot, err := fs.Sub(embedRules{root}, "static")
	if err != nil {
		panic(err) // fs.Sub only rejects a malformed subdirectory name; "static" isn't one
	}
	// fs.Sub doesn't stat anything, so a missing sourceDir/static wouldn't
	// fail here — it would 404 every asset at request time instead. The
	// caller is expected to have already checked the tree is real (see
	// cmd/server's devSourceDir).
	return pages{source: root}, noStore(http.FileServerFS(staticRoot))
}

// noStore keeps the browser from heuristically caching a file that's
// expected to change under it. The embedded handler never sends
// Last-Modified (embedded files carry a zero mtime), so browsers never
// heuristically cache them; the disk-backed handler sends a real one, which
// is exactly what invites that caching — an edited game.css would
// otherwise keep serving from cache and dev mode would look broken.
func noStore(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		h.ServeHTTP(w, r)
	})
}
