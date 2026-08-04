package web

import (
	"context"
	"net/http"

	"github.com/matt-in-space/diplomacy/application/auth"
)

const sessionCookieName = "session"

type contextKey int

const currentPlayerKey contextKey = iota

// withCurrentPlayer populates the request context with the current player
// if a valid session cookie is present. It never blocks a request — a
// missing or invalid cookie just means the request proceeds anonymously.
// A "require login, redirect otherwise" variant isn't needed yet; nothing
// currently requires being logged in to reach it.
func withCurrentPlayer(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err == nil {
				if player, err := authService.Authenticate(r.Context(), cookie.Value); err == nil {
					r = r.WithContext(context.WithValue(r.Context(), currentPlayerKey, player))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// currentPlayer returns the logged-in player for this request, if any.
func currentPlayer(r *http.Request) (*auth.Player, bool) {
	player, ok := r.Context().Value(currentPlayerKey).(*auth.Player)
	return player, ok
}
