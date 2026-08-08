package web_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/web"
)

func TestHomeRendersHelloPage(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t), newTestLobbyService(t))

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
	mux := web.NewMux(newTestAuthService(t), newTestLobbyService(t))

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
	if strings.Contains(body, "Create Game") {
		t.Fatalf("anonymous body should not show Create Game: %q", body)
	}
}

func TestHomeShowsDisplayNameWhenLoggedIn(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t), newTestLobbyService(t))

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

	body := rec.Body.String()
	if !strings.Contains(body, "Alice") {
		t.Fatalf("body missing display name: %q", body)
	}
	if !strings.Contains(body, "Create Game") {
		t.Fatalf("logged-in body should show Create Game: %q", body)
	}
}

func TestStaticServesPicoCSS(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t), newTestLobbyService(t))

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

func TestStaticServesWesternEuropeMapData(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t), newTestLobbyService(t))

	req := httptest.NewRequest(http.MethodGet, "/static/maps/western-europe-subset.json", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var data struct {
		MapID     string `json:"mapId"`
		Provinces map[string]struct {
			D       string    `json:"d"`
			LabelAt []float64 `json:"labelAt"`
			Coasts  map[string]struct {
				D string `json:"d"`
			} `json:"coasts"`
		} `json:"provinces"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if data.MapID != "western-europe-subset" {
		t.Fatalf("mapId = %q, want %q", data.MapID, "western-europe-subset")
	}

	// The real province IDs from core/gamemap/testdata/western_europe.json
	// — every one of them needs a visual entry, or the map can't fully
	// render.
	for _, id := range []string{"par", "bre", "gas", "mao", "eng", "lon", "spa", "por"} {
		province, ok := data.Provinces[id]
		if !ok {
			t.Fatalf("provinces missing entry for %q", id)
		}
		if province.D == "" {
			t.Fatalf("province %q has an empty path", id)
		}
	}

	spa := data.Provinces["spa"]
	for _, coast := range []string{"spa-nc", "spa-sc"} {
		if c, ok := spa.Coasts[coast]; !ok || c.D == "" {
			t.Fatalf("spa missing coast %q with a non-empty path", coast)
		}
	}
}

func TestUnknownStaticFileReturnsNotFound(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t), newTestLobbyService(t))

	req := httptest.NewRequest(http.MethodGet, "/static/missing.css", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
