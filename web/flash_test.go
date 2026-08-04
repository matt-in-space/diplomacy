package web

import (
	"net/http/httptest"
	"testing"
)

func TestSetFlashThenPopFlash(t *testing.T) {
	rec := httptest.NewRecorder()
	setFlash(rec, "error", "Invalid email or password.")

	req := httptest.NewRequest("GET", "/", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}

	popRec := httptest.NewRecorder()
	kind, message, ok := popFlash(popRec, req)
	if !ok {
		t.Fatal("expected popFlash to find a pending flash")
	}
	if kind != "error" {
		t.Fatalf("kind = %q, want %q", kind, "error")
	}
	if message != "Invalid email or password." {
		t.Fatalf("message = %q, want %q", message, "Invalid email or password.")
	}
}

func TestPopFlashClearsIt(t *testing.T) {
	setRec := httptest.NewRecorder()
	setFlash(setRec, "success", "Logged in.")

	req := httptest.NewRequest("GET", "/", nil)
	for _, c := range setRec.Result().Cookies() {
		req.AddCookie(c)
	}

	popRec := httptest.NewRecorder()
	if _, _, ok := popFlash(popRec, req); !ok {
		t.Fatal("expected popFlash to find a pending flash")
	}

	cleared := false
	for _, c := range popRec.Result().Cookies() {
		if c.Name == flashCookieName && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("expected popFlash to clear the flash cookie (MaxAge < 0)")
	}
}

func TestPopFlashWithNoCookieReturnsNotOK(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	if _, _, ok := popFlash(rec, req); ok {
		t.Fatal("expected popFlash to report no pending flash")
	}
}
