package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/web"
)

func TestHomeRendersHelloPage(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	if body := rec.Body.String(); !strings.Contains(body, "Hello, Diplomacy") {
		t.Fatalf("body = %q, want it to contain %q", body, "Hello, Diplomacy")
	}
}

func TestHomeShowsLoginSignupLinksWhenAnonymous(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `href="/login"`) {
		t.Fatalf("body missing login link: %q", body)
	}
	if !strings.Contains(body, `href="/signup"`) {
		t.Fatalf("body missing signup link: %q", body)
	}
}

func TestHomeShowsDisplayNameWhenLoggedIn(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t))

	signup(t, mux, "a@example.com", "Alice", "password123")
	loginResp := login(t, mux, "a@example.com", "password123")
	cookie := sessionCookie(loginResp)
	if cookie == nil {
		t.Fatal("expected a session cookie after login")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if body := rec.Body.String(); !strings.Contains(body, "Alice") {
		t.Fatalf("body missing display name: %q", body)
	}
}

func TestStaticServesPicoCSS(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t))

	req := httptest.NewRequest(http.MethodGet, "/static/pico.min.css", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected a non-empty CSS body")
	}
}

func TestUnknownStaticFileReturnsNotFound(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t))

	req := httptest.NewRequest(http.MethodGet, "/static/missing.css", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
