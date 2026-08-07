package web

import (
	"context"
	"net/http"
	"net/url"
	"strings"

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

// redirectIfAuthenticated sends an already-logged-in visitor to the home
// page instead of showing a login/signup form. Relies on withCurrentPlayer
// having already populated context (it wraps the whole mux), so it only
// checks context, not the cookie directly. Applied to the GET (view the
// form) routes only — not POST, which is only reachable by deliberately
// crafting a request outside the UI, not a real risk worth guarding.
func redirectIfAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := currentPlayer(r); ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireAuthentication redirects to /login?next=<this request's path> if
// there's no current player. The blocking counterpart to
// withCurrentPlayer's non-blocking style — the first routes that actually
// need one.
func requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := currentPlayer(r); !ok {
			q := url.Values{"next": {r.URL.RequestURI()}}
			http.Redirect(w, r, "/login?"+q.Encode(), http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// safeRedirectTarget validates a caller-supplied redirect path before it's
// used in an http.Redirect. It must be same-site and relative — starting
// with exactly one "/", never "//" (protocol-relative, still points
// off-site) or a full URL — otherwise a `next` value could be turned into
// an open redirect. Returns "/" for anything invalid or empty.
func safeRedirectTarget(path string) string {
	if path == "" || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return "/"
	}
	if u, err := url.Parse(path); err != nil || u.Scheme != "" || u.Host != "" {
		return "/"
	}
	return path
}
