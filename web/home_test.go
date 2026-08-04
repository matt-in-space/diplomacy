package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/web"
)

func TestHomeRendersHelloPage(t *testing.T) {
	mux := web.NewMux()

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

func TestStaticServesPicoCSS(t *testing.T) {
	mux := web.NewMux()

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
	mux := web.NewMux()

	req := httptest.NewRequest(http.MethodGet, "/static/missing.css", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
