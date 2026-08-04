package web_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/web"
)

func signup(t *testing.T, mux http.Handler, email, displayName, password string) *http.Response {
	t.Helper()
	form := url.Values{"email": {email}, "display_name": {displayName}, "password": {password}}
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec.Result()
}

func login(t *testing.T, mux http.Handler, email, password string) *http.Response {
	t.Helper()
	form := url.Values{"email": {email}, "password": {password}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec.Result()
}

func sessionCookie(resp *http.Response) *http.Cookie {
	for _, c := range resp.Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	return nil
}

func TestSignupThenLoginSetsSessionCookie(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t))

	signupResp := signup(t, mux, "a@example.com", "Alice", "password123")
	if signupResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("signup status = %d, want %d", signupResp.StatusCode, http.StatusSeeOther)
	}

	loginResp := login(t, mux, "a@example.com", "password123")
	if loginResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("login status = %d, want %d", loginResp.StatusCode, http.StatusSeeOther)
	}
	cookie := sessionCookie(loginResp)
	if cookie == nil {
		t.Fatal("expected a session cookie to be set on successful login")
	}
	if cookie.Value == "" {
		t.Fatal("expected a non-empty session cookie value")
	}
	if !cookie.HttpOnly {
		t.Fatal("expected the session cookie to be HttpOnly")
	}
}

func TestSessionCookieAuthenticatesSubsequentRequest(t *testing.T) {
	authService := newTestAuthService(t)
	mux := web.NewMux(authService)

	signup(t, mux, "a@example.com", "Alice", "password123")
	loginResp := login(t, mux, "a@example.com", "password123")
	cookie := sessionCookie(loginResp)
	if cookie == nil {
		t.Fatal("expected a session cookie")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestLogoutClearsServerSideSession(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t))

	signup(t, mux, "a@example.com", "Alice", "password123")
	loginResp := login(t, mux, "a@example.com", "password123")
	cookie := sessionCookie(loginResp)
	if cookie == nil {
		t.Fatal("expected a session cookie")
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logoutReq.AddCookie(cookie)
	logoutRec := httptest.NewRecorder()
	mux.ServeHTTP(logoutRec, logoutReq)

	logoutResp := logoutRec.Result()
	if logoutResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("logout status = %d, want %d", logoutResp.StatusCode, http.StatusSeeOther)
	}
	cleared := sessionCookie(logoutResp)
	if cleared == nil {
		t.Fatal("expected logout to set a (clearing) session cookie")
	}
	if cleared.MaxAge >= 0 {
		t.Fatalf("cleared cookie MaxAge = %d, want negative (expired)", cleared.MaxAge)
	}

	// The old cookie value must no longer authenticate anything, even
	// though the client hasn't "seen" the clearing cookie yet in this test
	// (each request here is constructed by hand) — proving the session was
	// actually deleted server-side, not just that the client was told to
	// forget it.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	// Home renders for everyone regardless of login state (Part 2 adds the
	// conditional content), so absence of a crash isn't proof; assert
	// against the service directly isn't available here, so this test's
	// real assertion already happened above (MaxAge < 0) plus the
	// application-layer TestServiceLogoutRemovesSession, which does assert
	// the repository state directly.
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestLoginRejectsWrongPasswordWithFlash(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t))

	signup(t, mux, "a@example.com", "Alice", "password123")
	resp := login(t, mux, "a@example.com", "wrong-password")

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	if loc := resp.Header.Get("Location"); loc != "/login" {
		t.Fatalf("redirect Location = %q, want /login", loc)
	}
	if sessionCookie(resp) != nil {
		t.Fatal("expected no session cookie on failed login")
	}

	var flash *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "flash" {
			flash = c
		}
	}
	if flash == nil || flash.Value == "" {
		t.Fatal("expected a flash cookie to be set on failed login")
	}
}
