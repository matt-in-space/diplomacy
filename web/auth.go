package web

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/matt-in-space/diplomacy/application/auth"
)

func handleSignupForm(tmpl pages) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := authFormPageData{
			pageData: newPageData(w, r),
			Next:     safeRedirectTarget(r.URL.Query().Get("next")),
		}
		t, err := tmpl.signup()
		if err == nil {
			err = t.ExecuteTemplate(w, "layout", data)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func handleSignupSubmit(authService *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		next := safeRedirectTarget(r.PostFormValue("next"))

		_, err := authService.Signup(r.Context(), r.PostFormValue("email"), r.PostFormValue("display_name"), r.PostFormValue("password"))
		if err != nil {
			setFlash(w, "error", err.Error())
			http.Redirect(w, r, "/signup"+nextQuery(next), http.StatusSeeOther)
			return
		}

		setFlash(w, "success", "Account created. Please log in.")
		http.Redirect(w, r, "/login"+nextQuery(next), http.StatusSeeOther)
	}
}

func handleLoginForm(tmpl pages) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := authFormPageData{
			pageData: newPageData(w, r),
			Next:     safeRedirectTarget(r.URL.Query().Get("next")),
		}
		t, err := tmpl.login()
		if err == nil {
			err = t.ExecuteTemplate(w, "layout", data)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func handleLoginSubmit(authService *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		next := safeRedirectTarget(r.PostFormValue("next"))

		session, err := authService.Login(r.Context(), r.PostFormValue("email"), r.PostFormValue("password"))
		if err != nil {
			message := "Something went wrong. Please try again."
			if errors.Is(err, auth.ErrInvalidCredentials) {
				message = "Invalid email or password."
			}
			setFlash(w, "error", message)
			http.Redirect(w, r, "/login"+nextQuery(next), http.StatusSeeOther)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    session.Token,
			Path:     "/",
			Expires:  session.ExpiresAt,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			// Not Secure yet — that requires knowing whether we're behind
			// TLS, which there's no dev/prod distinction for yet. Must be
			// set before any real deployment.
		})
		setFlash(w, "success", "Logged in.")
		http.Redirect(w, r, next, http.StatusSeeOther)
	}
}

// nextQuery renders a "?next=..." query string for appending to a redirect
// target, or "" if next is just the default "/" — no point cluttering
// every ordinary signup/login URL with a no-op query param.
func nextQuery(next string) string {
	if next == "" || next == "/" {
		return ""
	}
	return "?next=" + url.QueryEscape(next)
}

func handleLogout(authService *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			// Best-effort: DeleteSession is idempotent, and a failure here
			// shouldn't stop the user from being logged out client-side.
			_ = authService.Logout(r.Context(), cookie.Value)
		}

		http.SetCookie(w, &http.Cookie{
			Name:   sessionCookieName,
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
		setFlash(w, "success", "Logged out.")
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
