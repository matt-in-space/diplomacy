package web_test

import (
	"testing"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/application/lobby"
	"github.com/matt-in-space/diplomacy/core/gamemap"
	"github.com/matt-in-space/diplomacy/infrastructure/memory"
)

// newTestAuthService returns an auth.Service backed by fresh, empty
// in-memory repositories — isolated per test, same as every other test in
// this codebase that exercises real repository round-tripping rather than
// a call-counting fake.
func newTestAuthService(t *testing.T) *auth.Service {
	t.Helper()
	return auth.NewService(memory.NewPlayerRepository(), memory.NewSessionRepository())
}

// newTestLobbyService returns a lobby.Service backed by fresh in-memory
// repositories, with the real embedded Western Europe map pre-loaded —
// tests exercise the actual create-game form/handler round-trip against
// the same map ID production uses, not a synthetic stand-in. Also returns
// the *gameplay.GameplayService backing it (same repositories, so a game
// lobbyService starts is visible through it too) — web.NewMux needs both,
// same as cmd/server/main.go's real wiring.
func newTestLobbyService(t *testing.T) (*lobby.Service, *gameplay.GameplayService) {
	t.Helper()
	gm, err := gamemap.WesternEurope()
	if err != nil {
		t.Fatalf("WesternEurope failed: %v", err)
	}

	games := memory.NewGameRepository()
	maps := memory.NewGameMapRepository(gm)
	setups := memory.NewGameSetupRepository()
	gp := gameplay.NewGameplayService(games, maps)
	return lobby.NewService(setups, games, maps, gp), gp
}
