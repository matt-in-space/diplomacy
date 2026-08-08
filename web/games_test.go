package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/application/lobby"
	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/web"
)

// createGame submits the create-game form as cookie's player and returns
// the resulting lobby location and the setup's invite code, fetched
// straight from the service so tests assert against the real value.
func createGame(t *testing.T, mux http.Handler, lobbyService *lobby.Service, cookie *http.Cookie) (loc, code string) {
	t.Helper()
	form := url.Values{"map_id": {"western-europe-subset"}}
	req := httptest.NewRequest(http.MethodPost, "/games", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	loc = rec.Result().Header.Get("Location")
	id := game.GameID(strings.TrimSuffix(strings.TrimPrefix(loc, "/games/"), "/lobby"))
	setup, err := lobbyService.GetGameSetup(context.Background(), id)
	if err != nil {
		t.Fatalf("GetGameSetup failed: %v", err)
	}
	return loc, setup.InviteCode
}

// joinGame submits the join-by-code form as cookie's player.
func joinGame(t *testing.T, mux http.Handler, cookie *http.Cookie, code string) *http.Response {
	t.Helper()
	form := url.Values{"code": {code}}
	req := httptest.NewRequest(http.MethodPost, "/games/join", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec.Result()
}

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

func TestJoinGameAddsPlayerAndRedirectsToLobby(t *testing.T) {
	lobbyService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService)

	signup(t, mux, "host@example.com", "Hosty", "password123")
	hostCookie := sessionCookie(login(t, mux, "host@example.com", "password123"))
	if hostCookie == nil {
		t.Fatal("expected a session cookie for host")
	}
	loc, code := createGame(t, mux, lobbyService, hostCookie)

	signup(t, mux, "joiner@example.com", "Joiny", "password123")
	joinerCookie := sessionCookie(login(t, mux, "joiner@example.com", "password123"))
	if joinerCookie == nil {
		t.Fatal("expected a session cookie for joiner")
	}

	resp := joinGame(t, mux, joinerCookie, code)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	if got := resp.Header.Get("Location"); got != loc {
		t.Fatalf("Location = %q, want %q", got, loc)
	}

	req := httptest.NewRequest(http.MethodGet, loc, nil)
	req.AddCookie(joinerCookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "Hosty") || !strings.Contains(body, "Joiny") {
		t.Fatalf("lobby body missing both players: %q", body)
	}
	if !strings.Contains(body, "Players (2)") {
		t.Fatalf("lobby body missing updated player count: %q", body)
	}
}

func TestJoinGameIsIdempotent(t *testing.T) {
	lobbyService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService)

	signup(t, mux, "host@example.com", "Hosty", "password123")
	hostCookie := sessionCookie(login(t, mux, "host@example.com", "password123"))
	_, code := createGame(t, mux, lobbyService, hostCookie)

	signup(t, mux, "joiner@example.com", "Joiny", "password123")
	joinerCookie := sessionCookie(login(t, mux, "joiner@example.com", "password123"))

	first := joinGame(t, mux, joinerCookie, code)
	if first.StatusCode != http.StatusSeeOther {
		t.Fatalf("first join status = %d, want %d", first.StatusCode, http.StatusSeeOther)
	}
	second := joinGame(t, mux, joinerCookie, code)
	if second.StatusCode != http.StatusSeeOther {
		t.Fatalf("second join status = %d, want %d", second.StatusCode, http.StatusSeeOther)
	}
	if first.Header.Get("Location") != second.Header.Get("Location") {
		t.Fatalf("Location mismatch between joins: %q vs %q", first.Header.Get("Location"), second.Header.Get("Location"))
	}
}

func TestJoinGameRejectsUnknownCode(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t), newTestLobbyService(t))

	signup(t, mux, "a@example.com", "Alice", "password123")
	cookie := sessionCookie(login(t, mux, "a@example.com", "password123"))
	if cookie == nil {
		t.Fatal("expected a session cookie")
	}

	resp := joinGame(t, mux, cookie, "NOTREAL1")
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	if loc := resp.Header.Get("Location"); loc != "/games/join" {
		t.Fatalf("Location = %q, want /games/join", loc)
	}

	var flash *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "flash" {
			flash = c
		}
	}
	if flash == nil || flash.Value == "" {
		t.Fatal("expected a flash cookie for an unknown code")
	}

	// Follow the redirect, carrying the flash cookie, to confirm the actual
	// message rendered — not just that some flash was set — so a bug that
	// mapped every JoinGameSetup error to the same generic text wouldn't
	// slip through.
	followReq := httptest.NewRequest(http.MethodGet, "/games/join", nil)
	followReq.AddCookie(cookie)
	followReq.AddCookie(flash)
	followRec := httptest.NewRecorder()
	mux.ServeHTTP(followRec, followReq)
	if !strings.Contains(followRec.Body.String(), "doesn&#39;t match a game") {
		t.Fatalf("body missing unknown-code message: %q", followRec.Body.String())
	}
}

func TestJoinGameRejectsWhenFull(t *testing.T) {
	lobbyService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService)

	signup(t, mux, "host@example.com", "Hosty", "password123")
	hostCookie := sessionCookie(login(t, mux, "host@example.com", "password123"))
	_, code := createGame(t, mux, lobbyService, hostCookie)

	// western-europe-subset has 2 nations — host plus one joiner fills it.
	signup(t, mux, "joiner@example.com", "Joiny", "password123")
	joinerCookie := sessionCookie(login(t, mux, "joiner@example.com", "password123"))
	if resp := joinGame(t, mux, joinerCookie, code); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("joiner join status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}

	signup(t, mux, "third@example.com", "Thirdy", "password123")
	thirdCookie := sessionCookie(login(t, mux, "third@example.com", "password123"))
	resp := joinGame(t, mux, thirdCookie, code)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	if loc := resp.Header.Get("Location"); loc != "/games/join" {
		t.Fatalf("Location = %q, want /games/join", loc)
	}

	var flash *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "flash" {
			flash = c
		}
	}
	if flash == nil || flash.Value == "" {
		t.Fatal("expected a flash cookie when the game is full")
	}

	followReq := httptest.NewRequest(http.MethodGet, "/games/join", nil)
	followReq.AddCookie(thirdCookie)
	followReq.AddCookie(flash)
	followRec := httptest.NewRecorder()
	mux.ServeHTTP(followRec, followReq)
	if !strings.Contains(followRec.Body.String(), "already full") {
		t.Fatalf("body missing full-game message: %q", followRec.Body.String())
	}
}

func TestGamesJoinRequiresLogin(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t), newTestLobbyService(t))

	req := httptest.NewRequest(http.MethodGet, "/games/join", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	if loc := resp.Header.Get("Location"); loc != "/login?next=%2Fgames%2Fjoin" {
		t.Fatalf("Location = %q, want /login?next=%%2Fgames%%2Fjoin", loc)
	}
}

func TestLobbyStartButtonDisabledUntilFullThenEnabled(t *testing.T) {
	lobbyService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService)

	signup(t, mux, "host@example.com", "Hosty", "password123")
	hostCookie := sessionCookie(login(t, mux, "host@example.com", "password123"))
	loc, code := createGame(t, mux, lobbyService, hostCookie)

	fetchLobby := func() string {
		req := httptest.NewRequest(http.MethodGet, loc, nil)
		req.AddCookie(hostCookie)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Body.String()
	}

	// western-europe-subset has 2 nations — host alone is 1/2, not full yet.
	before := fetchLobby()
	if !strings.Contains(before, "Start Game (1/2)") {
		t.Fatalf("body missing 1/2 progress: %q", before)
	}
	if !strings.Contains(before, "disabled") {
		t.Fatalf("body should have a disabled Start button before the lobby is full: %q", before)
	}

	signup(t, mux, "joiner@example.com", "Joiny", "password123")
	joinerCookie := sessionCookie(login(t, mux, "joiner@example.com", "password123"))
	if resp := joinGame(t, mux, joinerCookie, code); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("join status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}

	after := fetchLobby()
	if !strings.Contains(after, "Start Game (2/2)") {
		t.Fatalf("body missing 2/2 progress: %q", after)
	}
	if strings.Contains(after, "disabled") {
		t.Fatalf("Start button should no longer be disabled once full: %q", after)
	}
}

func TestStartGameSucceedsWhenFullAndRedirectsToGame(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/games/"+gameID+"/start", nil)
	req.AddCookie(hostCookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	if got := resp.Header.Get("Location"); got != "/games/"+gameID {
		t.Fatalf("Location = %q, want /games/%s", got, gameID)
	}
}

func TestStartGameRejectsWhenNotFull(t *testing.T) {
	lobbyService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService)

	signup(t, mux, "host@example.com", "Hosty", "password123")
	hostCookie := sessionCookie(login(t, mux, "host@example.com", "password123"))
	loc, _ := createGame(t, mux, lobbyService, hostCookie)
	gameID := strings.TrimSuffix(strings.TrimPrefix(loc, "/games/"), "/lobby")

	req := httptest.NewRequest(http.MethodPost, "/games/"+gameID+"/start", nil)
	req.AddCookie(hostCookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	if got := resp.Header.Get("Location"); got != loc {
		t.Fatalf("Location = %q, want %q (back to the lobby)", got, loc)
	}

	var flash *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "flash" {
			flash = c
		}
	}
	if flash == nil {
		t.Fatal("expected a flash cookie when starting an unfull lobby")
	}

	followReq := httptest.NewRequest(http.MethodGet, loc, nil)
	followReq.AddCookie(hostCookie)
	followReq.AddCookie(flash)
	followRec := httptest.NewRecorder()
	mux.ServeHTTP(followRec, followReq)
	if !strings.Contains(followRec.Body.String(), "Waiting for all players") {
		t.Fatalf("body missing not-full message: %q", followRec.Body.String())
	}
}

func TestStartGameRejectsNonHost(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/games/"+gameID+"/start", nil)
	req.AddCookie(joinerCookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}

	var flash *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "flash" {
			flash = c
		}
	}
	if flash == nil {
		t.Fatal("expected a flash cookie for a non-host start attempt")
	}

	followReq := httptest.NewRequest(http.MethodGet, loc, nil)
	followReq.AddCookie(joinerCookie)
	followReq.AddCookie(flash)
	followRec := httptest.NewRecorder()
	mux.ServeHTTP(followRec, followReq)
	if !strings.Contains(followRec.Body.String(), "Only the host") {
		t.Fatalf("body missing non-host message: %q", followRec.Body.String())
	}
}

func TestGameRedirectsToLobbyWhilePending(t *testing.T) {
	lobbyService := newTestLobbyService(t)
	mux := web.NewMux(newTestAuthService(t), lobbyService)

	signup(t, mux, "host@example.com", "Hosty", "password123")
	hostCookie := sessionCookie(login(t, mux, "host@example.com", "password123"))
	loc, _ := createGame(t, mux, lobbyService, hostCookie)
	gameID := strings.TrimSuffix(strings.TrimPrefix(loc, "/games/"), "/lobby")

	req := httptest.NewRequest(http.MethodGet, "/games/"+gameID, nil)
	req.AddCookie(hostCookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	if got := resp.Header.Get("Location"); got != loc {
		t.Fatalf("Location = %q, want %q", got, loc)
	}
}

func TestGameRendersFrontendShellWhenActive(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/games/"+gameID, nil)
	req.AddCookie(hostCookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `data-game-id="`+gameID+`"`) {
		t.Fatalf("body missing the game id data attribute: %q", body)
	}
	if !strings.Contains(body, `data-map-id="western-europe-subset"`) {
		t.Fatalf("body missing the map id data attribute: %q", body)
	}
	if !strings.Contains(body, `<script type="module" src="/static/js/main.js">`) {
		t.Fatalf("body missing the frontend script tag: %q", body)
	}

	// The lobby page should now report Active status too.
	lobbyReq := httptest.NewRequest(http.MethodGet, loc, nil)
	lobbyReq.AddCookie(hostCookie)
	lobbyRec := httptest.NewRecorder()
	mux.ServeHTTP(lobbyRec, lobbyReq)
	if !strings.Contains(lobbyRec.Body.String(), "active") {
		t.Fatalf("lobby body missing active status: %q", lobbyRec.Body.String())
	}
}

func TestGameRequiresLogin(t *testing.T) {
	mux := web.NewMux(newTestAuthService(t), newTestLobbyService(t))

	req := httptest.NewRequest(http.MethodGet, "/games/some-id", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	if loc := resp.Header.Get("Location"); loc != "/login?next=%2Fgames%2Fsome-id" {
		t.Fatalf("Location = %q, want /login?next=%%2Fgames%%2Fsome-id", loc)
	}
}
