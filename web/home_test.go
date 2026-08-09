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
	if strings.Contains(body, "Your games") {
		t.Fatalf("anonymous body should not show a games list: %q", body)
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
	if strings.Contains(body, "Your games") {
		t.Fatalf("body should not show a games list with no games yet: %q", body)
	}
}

func TestHomeShowsPendingGameInList(t *testing.T) {
	lobbyService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService)

	signup(t, mux, "host@example.com", "Hosty", "password123")
	hostCookie := sessionCookie(login(t, mux, "host@example.com", "password123"))
	loc, _ := createGame(t, mux, lobbyService, hostCookie)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(hostCookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "Your games") {
		t.Fatalf("body missing the games list heading: %q", body)
	}
	if !strings.Contains(body, "Lobby, waiting for players") {
		t.Fatalf("body missing the pending-status text: %q", body)
	}
	if !strings.Contains(body, `href="`+loc+`"`) {
		t.Fatalf("body missing a link to %q: %q", loc, body)
	}
}

func TestHomeShowsActiveGameWithFormattedTurn(t *testing.T) {
	lobbyService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService)

	signup(t, mux, "host@example.com", "Hosty", "password123")
	hostCookie := sessionCookie(login(t, mux, "host@example.com", "password123"))
	loc, code := createGame(t, mux, lobbyService, hostCookie)
	gameID := strings.TrimSuffix(strings.TrimPrefix(loc, "/games/"), "/lobby")

	signup(t, mux, "joiner@example.com", "Joiny", "password123")
	joinerCookie := sessionCookie(login(t, mux, "joiner@example.com", "password123"))
	if resp := joinGame(t, mux, joinerCookie, code); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("join status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}

	startReq := httptest.NewRequest(http.MethodPost, "/games/"+gameID+"/start", nil)
	startReq.AddCookie(hostCookie)
	startRec := httptest.NewRecorder()
	mux.ServeHTTP(startRec, startReq)
	if startRec.Result().StatusCode != http.StatusSeeOther {
		t.Fatalf("start status = %d, want %d", startRec.Result().StatusCode, http.StatusSeeOther)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(hostCookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "Spring 1, awaiting orders") {
		t.Fatalf("body missing the formatted turn text: %q", body)
	}
	if !strings.Contains(body, `href="/games/`+gameID+`"`) {
		t.Fatalf("body missing a link to /games/%s: %q", gameID, body)
	}
}

func TestHomeListsGamesNewestFirst(t *testing.T) {
	lobbyService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService)

	signup(t, mux, "host@example.com", "Hosty", "password123")
	hostCookie := sessionCookie(login(t, mux, "host@example.com", "password123"))

	firstLoc, _ := createGame(t, mux, lobbyService, hostCookie)
	secondLoc, _ := createGame(t, mux, lobbyService, hostCookie)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(hostCookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	firstIdx := strings.Index(body, `href="`+firstLoc+`"`)
	secondIdx := strings.Index(body, `href="`+secondLoc+`"`)
	if firstIdx == -1 || secondIdx == -1 {
		t.Fatalf("body missing a link to one of the created games: %q", body)
	}
	if secondIdx > firstIdx {
		t.Fatalf("expected the second (newer) game to render before the first: %q", body)
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
			Name         string    `json:"name"`
			Type         string    `json:"type"`
			SupplyCenter bool      `json:"supplyCenter"`
			D            string    `json:"d"`
			LabelAt      []float64 `json:"labelAt"`
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
	knownTypes := map[string]bool{"inland": true, "coastal": true, "water": true}
	for _, id := range []string{"par", "bre", "gas", "mao", "eng", "lon", "spa", "por"} {
		province, ok := data.Provinces[id]
		if !ok {
			t.Fatalf("provinces missing entry for %q", id)
		}
		if province.D == "" {
			t.Fatalf("province %q has an empty path", id)
		}
		if province.Name == "" {
			t.Fatalf("province %q has an empty name", id)
		}
		if !knownTypes[province.Type] {
			t.Fatalf("province %q has an unknown type %q", id, province.Type)
		}
	}

	// Supply centers per core/gamemap/testdata/western_europe.json — this
	// is what drives the thicker supply-center border in game.css.
	for _, id := range []string{"par", "bre", "lon", "spa", "por"} {
		if !data.Provinces[id].SupplyCenter {
			t.Fatalf("province %q should be a supply center", id)
		}
	}
	for _, id := range []string{"gas", "mao", "eng"} {
		if data.Provinces[id].SupplyCenter {
			t.Fatalf("province %q should not be a supply center", id)
		}
	}
}

func TestStaticServesGameCSS(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t), newTestLobbyService(t))

	req := httptest.NewRequest(http.MethodGet, "/static/game.css", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected a non-empty CSS body")
	}
}

func TestStaticServesMainModule(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t), newTestLobbyService(t))

	req := httptest.NewRequest(http.MethodGet, "/static/js/main.mjs", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected a non-empty script body")
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/javascript") {
		t.Fatalf("Content-Type = %q, want text/javascript", ct)
	}
}

// TestStaticDoesNotServeJSTests pins the go:embed underscore-exclusion
// behavior that keeps web/static/js/_tests/ (test-only code) out of the
// embedded binary and off the public static route. If _tests/ were ever
// renamed to something not starting with '_' or '.', this would start
// failing as the file becomes servable.
func TestStaticDoesNotServeJSTests(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t), newTestLobbyService(t))

	req := httptest.NewRequest(http.MethodGet, "/static/js/_tests/main.test.mjs", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
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
