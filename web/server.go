package web

import (
	"io/fs"
	"net/http"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/application/lobby"
)

// NewMux returns the full application handler: routes plus the global
// middleware chain. It's an http.Handler, not a *http.ServeMux, because
// once wrapped in middleware it genuinely isn't a bare mux anymore — and
// nothing (callers, tests) needs it to be, since ServeHTTP is all
// http.ListenAndServe or httptest.NewRecorder-based tests ever call.
func NewMux(authService *auth.Service, lobbyService *lobby.Service, gameplayService *gameplay.GameplayService) http.Handler {
	mux := http.NewServeMux()

	staticRoot, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err) // embedded at build time; a failure here is a build bug
	}

	mux.Handle("GET /", handleHome(lobbyService))
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticRoot)))

	mux.Handle("GET /signup", redirectIfAuthenticated(http.HandlerFunc(handleSignupForm)))
	mux.HandleFunc("POST /signup", handleSignupSubmit(authService))
	mux.Handle("GET /login", redirectIfAuthenticated(http.HandlerFunc(handleLoginForm)))
	mux.HandleFunc("POST /login", handleLoginSubmit(authService))
	mux.HandleFunc("POST /logout", handleLogout(authService))

	mux.Handle("GET /games/new", requireAuthentication(http.HandlerFunc(handleNewGameForm)))
	mux.Handle("POST /games", requireAuthentication(handleCreateGame(lobbyService)))
	mux.Handle("GET /games/join", requireAuthentication(http.HandlerFunc(handleJoinGameForm)))
	mux.Handle("POST /games/join", requireAuthentication(handleJoinGameSubmit(lobbyService)))
	mux.Handle("GET /games/{id}/lobby", requireAuthentication(handleGameSetupLobby(lobbyService, authService)))
	mux.Handle("POST /games/{id}/start", requireAuthentication(handleStartGame(lobbyService)))
	mux.Handle("GET /games/{id}", requireAuthentication(handleGame(lobbyService)))
	mux.Handle("GET /games/{id}/state", requireAuthentication(handleGameState(lobbyService, gameplayService)))

	// Global: more than just auth routes will eventually want to know who's
	// asking (nav, later pages). Never blocks a request — see
	// withCurrentPlayer's doc comment.
	return withCurrentPlayer(authService)(mux)
}
