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
	lobbyService, gameplayService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService, gameplayService)

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
	lobbyService, gameplayService := newTestLobbyService(t)
	mux := web.NewMux(authService, lobbyService, gameplayService)

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
	lobbyService, gameplayService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService, gameplayService)

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
	// forget it. Home now renders differently depending on login state, so
	// this can assert the stronger thing directly: the display name is gone.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if strings.Contains(rec.Body.String(), "Alice") {
		t.Fatal("expected the display name to be gone from the page after logout")
	}
}

func TestSignupFormRenders(t *testing.T) {
	lobbyService, gameplayService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService, gameplayService)

	req := httptest.NewRequest(http.MethodGet, "/signup", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `action="/signup"`) {
		t.Fatalf("body missing signup form action: %q", body)
	}
	if !strings.Contains(body, `name="display_name"`) {
		t.Fatalf("body missing display_name field: %q", body)
	}
}

func TestLoginFormRenders(t *testing.T) {
	lobbyService, gameplayService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService, gameplayService)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, `action="/login"`) {
		t.Fatalf("body missing login form action: %q", body)
	}
}

func TestSignupAndLoginFormsRedirectWhenAlreadyAuthenticated(t *testing.T) {
	lobbyService, gameplayService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService, gameplayService)

	signup(t, mux, "a@example.com", "Alice", "password123")
	loginResp := login(t, mux, "a@example.com", "password123")
	cookie := sessionCookie(loginResp)
	if cookie == nil {
		t.Fatal("expected a session cookie")
	}

	for _, path := range []string{"/signup", "/login"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("%s status = %d, want %d", path, rec.Code, http.StatusSeeOther)
		}
		if loc := rec.Header().Get("Location"); loc != "/" {
			t.Fatalf("%s redirect Location = %q, want /", path, loc)
		}
	}
}

func TestFlashRendersThroughRealTemplateThenClears(t *testing.T) {
	lobbyService, gameplayService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService, gameplayService)

	// No account exists yet, so this fails with a flash set.
	loginResp := login(t, mux, "missing@example.com", "password123")
	flash := loginResp.Cookies()
	var flashCookie *http.Cookie
	for _, c := range flash {
		if c.Name == "flash" {
			flashCookie = c
		}
	}
	if flashCookie == nil {
		t.Fatal("expected a flash cookie after a failed login")
	}

	// Following the redirect, carrying the flash cookie, the way a browser
	// would.
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.AddCookie(flashCookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "Invalid email or password.") {
		t.Fatalf("body missing flash message: %q", rec.Body.String())
	}

	var cleared *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "flash" {
			cleared = c
		}
	}
	if cleared == nil || cleared.MaxAge >= 0 {
		t.Fatal("expected the flash cookie to be cleared after rendering")
	}

	// A second GET /login with no flash cookie (matching a browser that
	// already consumed it) shows no message.
	req2 := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if strings.Contains(rec2.Body.String(), "Invalid email or password.") {
		t.Fatal("expected the flash message not to reappear without the flash cookie")
	}
}

func TestLoginRejectsWrongPasswordWithFlash(t *testing.T) {
	lobbyService, gameplayService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService, gameplayService)

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
