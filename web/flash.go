package web

import (
	"net/http"
	"net/url"
	"strings"
)

// Flash messages are independent of session/login state — they need to work
// for anonymous flows too ("account created, please log in" happens before
// any session exists), so they don't live on auth.Session. A single cookie
// carries kind and message across exactly one redirect; popFlash clears it
// immediately so it only ever shows once.
const flashCookieName = "flash"

func setFlash(w http.ResponseWriter, kind, message string) {
	// Cookie values have a restricted character set (RFC 6265) — no spaces,
	// commas, semicolons, etc. — so the message can't be written in raw;
	// query-escape the whole encoded pair.
	http.SetCookie(w, &http.Cookie{
		Name:     flashCookieName,
		Value:    url.QueryEscape(kind + "|" + message),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// popFlash returns the pending flash message, if any, and clears it so a
// page refresh doesn't show it again.
func popFlash(w http.ResponseWriter, r *http.Request) (kind, message string, ok bool) {
	c, err := r.Cookie(flashCookieName)
	if err != nil || c.Value == "" {
		return "", "", false
	}

	http.SetCookie(w, &http.Cookie{
		Name:   flashCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	decoded, err := url.QueryUnescape(c.Value)
	if err != nil {
		return "", "", false
	}
	k, msg, found := strings.Cut(decoded, "|")
	if !found {
		return "", "", false
	}
	return k, msg, true
}
