package web

import (
	"io/fs"
	"net/http"

	"github.com/matt-in-space/diplomacy/application/auth"
)

// NewMux returns the full application handler: routes plus the global
// middleware chain. It's an http.Handler, not a *http.ServeMux, because
// once wrapped in middleware it genuinely isn't a bare mux anymore — and
// nothing (callers, tests) needs it to be, since ServeHTTP is all
// http.ListenAndServe or httptest.NewRecorder-based tests ever call.
func NewMux(authService *auth.Service) http.Handler {
	mux := http.NewServeMux()

	staticRoot, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err) // embedded at build time; a failure here is a build bug
	}

	mux.HandleFunc("GET /", handleHome)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticRoot)))

	mux.Handle("GET /signup", redirectIfAuthenticated(http.HandlerFunc(handleSignupForm)))
	mux.HandleFunc("POST /signup", handleSignupSubmit(authService))
	mux.Handle("GET /login", redirectIfAuthenticated(http.HandlerFunc(handleLoginForm)))
	mux.HandleFunc("POST /login", handleLoginSubmit(authService))
	mux.HandleFunc("POST /logout", handleLogout(authService))

	// Global: more than just auth routes will eventually want to know who's
	// asking (nav, later pages). Never blocks a request — see
	// withCurrentPlayer's doc comment.
	return withCurrentPlayer(authService)(mux)
}
