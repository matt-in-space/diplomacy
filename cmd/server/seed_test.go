package main

import (
	"context"
	"testing"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/application/lobby"
	"github.com/matt-in-space/diplomacy/core/gamemap"
	"github.com/matt-in-space/diplomacy/infrastructure/memory"
)

func TestSeedDevUsersCreatesLoginableAccounts(t *testing.T) {
	authService := auth.NewService(memory.NewPlayerRepository(), memory.NewSessionRepository())
	ctx := context.Background()

	seedDevUsers(ctx, authService)

	for _, email := range []string{"user1@example.com", "user2@example.com"} {
		if _, err := authService.Login(ctx, email, "password"); err != nil {
			t.Fatalf("Login with seeded credentials for %s failed: %v", email, err)
		}
	}
}

func TestSeedDevUsersReturnsCreatedPlayers(t *testing.T) {
	authService := auth.NewService(memory.NewPlayerRepository(), memory.NewSessionRepository())
	ctx := context.Background()

	players := seedDevUsers(ctx, authService)
	if len(players) != 2 {
		t.Fatalf("len(players) = %d, want 2", len(players))
	}
	if players[0].Email != "user1@example.com" {
		t.Fatalf("players[0].Email = %q, want %q", players[0].Email, "user1@example.com")
	}
	if players[1].Email != "user2@example.com" {
		t.Fatalf("players[1].Email = %q, want %q", players[1].Email, "user2@example.com")
	}
}

func TestSeedDevUsersDoesNotPanicOnDuplicateCall(t *testing.T) {
	authService := auth.NewService(memory.NewPlayerRepository(), memory.NewSessionRepository())
	ctx := context.Background()

	seedDevUsers(ctx, authService)
	seedDevUsers(ctx, authService) // duplicate emails — logs and returns, doesn't panic or crash
}

func newTestLobbyServiceForSeed(t *testing.T) *lobby.Service {
	t.Helper()
	gm, err := gamemap.WesternEurope()
	if err != nil {
		t.Fatalf("WesternEurope failed: %v", err)
	}
	games := memory.NewGameRepository()
	maps := memory.NewGameMapRepository(gm)
	setups := memory.NewGameSetupRepository()
	gp := gameplay.NewGameplayService(games, maps)
	return lobby.NewService(setups, games, maps, gp)
}

func TestSeedDevGameCreatesJoinableLobby(t *testing.T) {
	authService := auth.NewService(memory.NewPlayerRepository(), memory.NewSessionRepository())
	lobbyService := newTestLobbyServiceForSeed(t)
	ctx := context.Background()

	players := seedDevUsers(ctx, authService)
	setup := seedDevGame(ctx, lobbyService, players)
	if setup == nil {
		t.Fatal("seedDevGame returned nil, want a created setup")
	}

	// Re-fetch through the service too, confirming what got persisted
	// matches what was returned.
	stored, err := lobbyService.GetGameSetup(ctx, setup.ID)
	if err != nil {
		t.Fatalf("GetGameSetup failed: %v", err)
	}
	if stored.HostID != players[0].ID {
		t.Fatalf("HostID = %q, want %q", setup.HostID, players[0].ID)
	}
	if setup.MapID != gamemap.WesternEuropeMapID {
		t.Fatalf("MapID = %q, want %q", setup.MapID, gamemap.WesternEuropeMapID)
	}
	if len(setup.PlayerIDs) != 2 || setup.PlayerIDs[0] != players[0].ID || setup.PlayerIDs[1] != players[1].ID {
		t.Fatalf("PlayerIDs = %v, want [%q %q]", setup.PlayerIDs, players[0].ID, players[1].ID)
	}
}

func TestSeedDevGameSkipsWithoutEnoughPlayers(t *testing.T) {
	lobbyService := newTestLobbyServiceForSeed(t)

	// Should log and return, not panic, with fewer than 2 players.
	seedDevGame(context.Background(), lobbyService, nil)
}
