package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/web"
)

func TestCreateGameRedirectsToLobby(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t), newTestLobbyService(t))

	signup(t, mux, "a@example.com", "Alice", "password123")
	loginResp := login(t, mux, "a@example.com", "password123")
	cookie := sessionCookie(loginResp)
	if cookie == nil {
		t.Fatal("expected a session cookie")
	}

	form := url.Values{"map_id": {"western-europe-subset"}}
	req := httptest.NewRequest(http.MethodPost, "/games", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	loc := resp.Header.Get("Location")
	if !strings.HasPrefix(loc, "/games/") || !strings.HasSuffix(loc, "/lobby") {
		t.Fatalf("Location = %q, want /games/{id}/lobby", loc)
	}
}

func TestGameSetupLobbyRendersStatusHostCodeAndPlayers(t *testing.T) {
	lobbyService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService)

	signup(t, mux, "a@example.com", "Alice", "password123")
	loginResp := login(t, mux, "a@example.com", "password123")
	cookie := sessionCookie(loginResp)
	if cookie == nil {
		t.Fatal("expected a session cookie")
	}

	form := url.Values{"map_id": {"western-europe-subset"}}
	createReq := httptest.NewRequest(http.MethodPost, "/games", strings.NewReader(form.Encode()))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createReq.AddCookie(cookie)
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, createReq)
	loc := createRec.Result().Header.Get("Location")

	// loc is "/games/{id}/lobby" — pull the real, randomly-generated
	// InviteCode straight from the service so the test asserts against the
	// actual value rendered, not just that some code-shaped text exists.
	id := game.GameID(strings.TrimSuffix(strings.TrimPrefix(loc, "/games/"), "/lobby"))
	setup, err := lobbyService.GetGameSetup(context.Background(), id)
	if err != nil {
		t.Fatalf("GetGameSetup failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, loc, nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "pending") {
		t.Fatalf("body missing pending status: %q", body)
	}
	if !strings.Contains(body, "You are the host") {
		t.Fatalf("body missing host notice: %q", body)
	}
	if !strings.Contains(body, setup.InviteCode) {
		t.Fatalf("body missing invite code %q: %q", setup.InviteCode, body)
	}
	if !strings.Contains(body, "Players (1)") {
		t.Fatalf("body missing player count heading: %q", body)
	}
	if !strings.Contains(body, "<table") || !strings.Contains(body, "Host (you)") {
		t.Fatalf("body missing players table with host row: %q", body)
	}
	if !strings.Contains(body, "Alice") {
		t.Fatalf("body missing display name Alice: %q", body)
	}
	if strings.Contains(body, string(setup.PlayerIDs[0])) {
		t.Fatalf("body leaks raw PlayerID %q, should show a display name instead: %q", setup.PlayerIDs[0], body)
	}
}

func TestCreateGameRejectsUnknownMap(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t), newTestLobbyService(t))

	signup(t, mux, "a@example.com", "Alice", "password123")
	loginResp := login(t, mux, "a@example.com", "password123")
	cookie := sessionCookie(loginResp)
	if cookie == nil {
		t.Fatal("expected a session cookie")
	}

	form := url.Values{"map_id": {"not-a-real-map"}}
	req := httptest.NewRequest(http.MethodPost, "/games", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	if loc := resp.Header.Get("Location"); loc != "/games/new" {
		t.Fatalf("Location = %q, want /games/new", loc)
	}

	var flash *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "flash" {
			flash = c
		}
	}
	if flash == nil || flash.Value == "" {
		t.Fatal("expected a flash cookie to be set on a rejected map")
	}
}

func TestGamesNewRequiresLoginAndRoundTripsBack(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t), newTestLobbyService(t))

	req := httptest.NewRequest(http.MethodGet, "/games/new", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	loc := resp.Header.Get("Location")
	if loc != "/login?next=%2Fgames%2Fnew" {
		t.Fatalf("Location = %q, want /login?next=%%2Fgames%%2Fnew", loc)
	}

	// Following that redirect, the login form should carry the same next
	// value in a hidden field, so submitting it lands back on /games/new.
	loginFormReq := httptest.NewRequest(http.MethodGet, loc, nil)
	loginFormRec := httptest.NewRecorder()
	mux.ServeHTTP(loginFormRec, loginFormReq)
	if !strings.Contains(loginFormRec.Body.String(), `name="next" value="/games/new"`) {
		t.Fatalf("login form missing next hidden field: %q", loginFormRec.Body.String())
	}

	signup(t, mux, "a@example.com", "Alice", "password123")
	form := url.Values{"email": {"a@example.com"}, "password": {"password123"}, "next": {"/games/new"}}
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRec := httptest.NewRecorder()
	mux.ServeHTTP(loginRec, loginReq)

	if got := loginRec.Result().Header.Get("Location"); got != "/games/new" {
		t.Fatalf("post-login redirect = %q, want /games/new", got)
	}
}

func TestSignupCarriesNextThroughToLogin(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t), newTestLobbyService(t))

	form := url.Values{
		"email":        {"a@example.com"},
		"display_name": {"Alice"},
		"password":     {"password123"},
		"next":         {"/games/new"},
	}
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	loc := rec.Result().Header.Get("Location")
	if loc != "/login?next=%2Fgames%2Fnew" {
		t.Fatalf("Location = %q, want /login?next=%%2Fgames%%2Fnew", loc)
	}
}
