package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matt-in-space/diplomacy/application/lobby"
	"github.com/matt-in-space/diplomacy/core/game"
)

func testGameSetup(id game.GameID, hostID game.PlayerID, code string) *lobby.GameSetup {
	return &lobby.GameSetup{
		ID:         id,
		MapID:      "test-map",
		HostID:     hostID,
		InviteCode: code,
		PlayerIDs:  []game.PlayerID{hostID},
		CreatedAt:  time.Unix(0, 0).UTC(),
	}
}

func TestGameSetupRepositoryCreateAndGet(t *testing.T) {
	repo := NewGameSetupRepository()
	setup := testGameSetup("game-a", "player-a", "code-a")

	if err := repo.CreateGameSetup(context.Background(), setup); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stored, err := repo.GetGameSetup(context.Background(), setup.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if stored.HostID != setup.HostID {
		t.Fatalf("HostID = %q, want %q", stored.HostID, setup.HostID)
	}
	if len(stored.PlayerIDs) != 1 || stored.PlayerIDs[0] != "player-a" {
		t.Fatalf("PlayerIDs = %v, want [player-a]", stored.PlayerIDs)
	}
}

func TestGameSetupRepositoryGetRejectsUnknownSetup(t *testing.T) {
	repo := NewGameSetupRepository()

	_, err := repo.GetGameSetup(context.Background(), "missing-game")
	if !errors.Is(err, lobby.ErrGameSetupNotFound) {
		t.Fatalf("Get error = %v, want ErrGameSetupNotFound", err)
	}
}

func TestGameSetupRepositoryRejectsDuplicateID(t *testing.T) {
	repo := NewGameSetupRepository()
	ctx := context.Background()

	if err := repo.CreateGameSetup(ctx, testGameSetup("game-a", "player-a", "code-a")); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	err := repo.CreateGameSetup(ctx, testGameSetup("game-a", "player-b", "code-b"))
	if !errors.Is(err, lobby.ErrGameSetupAlreadyExists) {
		t.Fatalf("second Create error = %v, want ErrGameSetupAlreadyExists", err)
	}
}

func TestGameSetupRepositoryGetByInviteCode(t *testing.T) {
	repo := NewGameSetupRepository()
	ctx := context.Background()
	setup := testGameSetup("game-a", "player-a", "code-a")

	if err := repo.CreateGameSetup(ctx, setup); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stored, err := repo.GetGameSetupByInviteCode(ctx, "code-a")
	if err != nil {
		t.Fatalf("GetGameSetupByInviteCode failed: %v", err)
	}
	if stored.ID != setup.ID {
		t.Fatalf("ID = %q, want %q", stored.ID, setup.ID)
	}
}

func TestGameSetupRepositoryGetByInviteCodeRejectsUnknownCode(t *testing.T) {
	repo := NewGameSetupRepository()

	_, err := repo.GetGameSetupByInviteCode(context.Background(), "missing-code")
	if !errors.Is(err, lobby.ErrGameSetupNotFound) {
		t.Fatalf("GetGameSetupByInviteCode error = %v, want ErrGameSetupNotFound", err)
	}
}

func TestGameSetupRepositorySaveExistingSetup(t *testing.T) {
	repo := NewGameSetupRepository()
	setup := testGameSetup("game-a", "player-a", "code-a")
	ctx := context.Background()

	if err := repo.CreateGameSetup(ctx, setup); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	cancelledAt := time.Unix(100, 0).UTC()
	setup.CancelledAt = &cancelledAt
	if err := repo.SaveGameSetup(ctx, setup); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	stored, err := repo.GetGameSetup(ctx, setup.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if stored.CancelledAt == nil || !stored.CancelledAt.Equal(cancelledAt) {
		t.Fatalf("CancelledAt = %v, want %v", stored.CancelledAt, cancelledAt)
	}
}

func TestGameSetupRepositorySaveRejectsUnknownSetup(t *testing.T) {
	repo := NewGameSetupRepository()

	err := repo.SaveGameSetup(context.Background(), testGameSetup("missing-game", "player-a", "code-a"))
	if !errors.Is(err, lobby.ErrGameSetupNotFound) {
		t.Fatalf("Save error = %v, want ErrGameSetupNotFound", err)
	}
}

func TestGameSetupRepositoryAddPlayerAppends(t *testing.T) {
	repo := NewGameSetupRepository()
	ctx := context.Background()
	setup := testGameSetup("game-a", "host-a", "code-a") // PlayerIDs: [host-a]

	if err := repo.CreateGameSetup(ctx, setup); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := repo.AddPlayerToGameSetup(ctx, "game-a", "player-b", 3); err != nil {
		t.Fatalf("AddPlayerToGameSetup failed: %v", err)
	}

	stored, err := repo.GetGameSetup(ctx, "game-a")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(stored.PlayerIDs) != 2 || stored.PlayerIDs[1] != "player-b" {
		t.Fatalf("PlayerIDs = %v, want [host-a player-b]", stored.PlayerIDs)
	}
}

func TestGameSetupRepositoryAddPlayerIsIdempotent(t *testing.T) {
	repo := NewGameSetupRepository()
	ctx := context.Background()
	setup := testGameSetup("game-a", "host-a", "code-a") // PlayerIDs: [host-a]

	if err := repo.CreateGameSetup(ctx, setup); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := repo.AddPlayerToGameSetup(ctx, "game-a", "host-a", 3); err != nil {
		t.Fatalf("AddPlayerToGameSetup(host) failed: %v", err)
	}

	stored, err := repo.GetGameSetup(ctx, "game-a")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(stored.PlayerIDs) != 1 {
		t.Fatalf("PlayerIDs = %v, want [host-a] unchanged", stored.PlayerIDs)
	}
}

func TestGameSetupRepositoryAddPlayerRejectsWhenFull(t *testing.T) {
	repo := NewGameSetupRepository()
	ctx := context.Background()
	setup := testGameSetup("game-a", "host-a", "code-a") // PlayerIDs: [host-a]

	if err := repo.CreateGameSetup(ctx, setup); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := repo.AddPlayerToGameSetup(ctx, "game-a", "player-b", 2); err != nil {
		t.Fatalf("AddPlayerToGameSetup(player-b) failed: %v", err)
	}

	err := repo.AddPlayerToGameSetup(ctx, "game-a", "player-c", 2)
	if !errors.Is(err, lobby.ErrGameSetupFull) {
		t.Fatalf("AddPlayerToGameSetup(player-c) error = %v, want ErrGameSetupFull", err)
	}
}

func TestGameSetupRepositoryAddPlayerRejectsUnknownSetup(t *testing.T) {
	repo := NewGameSetupRepository()

	err := repo.AddPlayerToGameSetup(context.Background(), "missing-game", "player-a", 3)
	if !errors.Is(err, lobby.ErrGameSetupNotFound) {
		t.Fatalf("AddPlayerToGameSetup error = %v, want ErrGameSetupNotFound", err)
	}
}

func TestGameSetupRepositoryListGameSetupsForPlayer(t *testing.T) {
	repo := NewGameSetupRepository()
	ctx := context.Background()

	// Distinct CreatedAt values (testGameSetup's default is identical
	// across every call) so the newest-first sort is actually exercised,
	// not just membership.
	setupA := testGameSetup("game-a", "player-a", "code-a")
	setupA.CreatedAt = time.Unix(100, 0).UTC()
	if err := repo.CreateGameSetup(ctx, setupA); err != nil {
		t.Fatalf("Create game-a failed: %v", err)
	}
	setupB := testGameSetup("game-b", "player-c", "code-b")
	setupB.CreatedAt = time.Unix(200, 0).UTC()
	if err := repo.CreateGameSetup(ctx, setupB); err != nil {
		t.Fatalf("Create game-b failed: %v", err)
	}
	setupC := testGameSetup("game-c", "player-b", "code-c")
	setupC.CreatedAt = time.Unix(300, 0).UTC()
	if err := repo.CreateGameSetup(ctx, setupC); err != nil {
		t.Fatalf("Create game-c failed: %v", err)
	}
	// player-a joins game-b as a non-host — ListGameSetupsForPlayer should
	// surface both games they host and games they merely joined.
	if err := repo.AddPlayerToGameSetup(ctx, "game-b", "player-a", 3); err != nil {
		t.Fatalf("AddPlayerToGameSetup failed: %v", err)
	}

	setups, err := repo.ListGameSetupsForPlayer(ctx, "player-a")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(setups) != 2 {
		t.Fatalf("len(setups) = %d, want 2", len(setups))
	}
	// Newest first: game-b (CreatedAt 200) before game-a (CreatedAt 100).
	if setups[0].ID != "game-b" || setups[1].ID != "game-a" {
		t.Fatalf("setups = [%q %q], want [game-b game-a] (newest first)", setups[0].ID, setups[1].ID)
	}
}

func TestGameSetupRepositoryStoresDetachedValues(t *testing.T) {
	repo := NewGameSetupRepository()
	ctx := context.Background()
	setup := testGameSetup("game-a", "host-a", "code-a")
	cancelledAt := time.Unix(100, 0).UTC()
	want := cancelledAt // a value copy, so mutating *setup.CancelledAt below can't move this
	setup.CancelledAt = &cancelledAt

	if err := repo.CreateGameSetup(ctx, setup); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	*setup.CancelledAt = time.Unix(200, 0).UTC() // mutate the caller's pointee in place
	setup.PlayerIDs[0] = "tampered"              // mutate the caller's backing array in place

	stored, err := repo.GetGameSetup(ctx, "game-a")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !stored.CancelledAt.Equal(want) {
		t.Fatalf("stored CancelledAt after source mutation = %v, want %v", stored.CancelledAt, want)
	}
	if stored.PlayerIDs[0] != "host-a" {
		t.Fatalf("stored PlayerIDs after source mutation = %v, want [host-a]", stored.PlayerIDs)
	}

	*stored.CancelledAt = time.Unix(300, 0).UTC() // mutate the fetched copy's pointee in place
	stored.PlayerIDs[0] = "tampered-again"        // mutate the fetched copy's backing array in place
	again, err := repo.GetGameSetup(ctx, "game-a")
	if err != nil {
		t.Fatalf("second Get failed: %v", err)
	}
	if !again.CancelledAt.Equal(want) {
		t.Fatalf("stored CancelledAt after fetched copy mutation = %v, want %v", again.CancelledAt, want)
	}
	if again.PlayerIDs[0] != "host-a" {
		t.Fatalf("stored PlayerIDs after fetched copy mutation = %v, want [host-a]", again.PlayerIDs)
	}
}

func TestGameSetupRepositoryRejectsNilSetup(t *testing.T) {
	repo := NewGameSetupRepository()
	ctx := context.Background()

	if err := repo.CreateGameSetup(ctx, nil); err == nil {
		t.Fatal("expected Create to reject nil setup")
	}
	if err := repo.SaveGameSetup(ctx, nil); err == nil {
		t.Fatal("expected Save to reject nil setup")
	}
}

func TestGameSetupRepositoryHonorsCancelledContext(t *testing.T) {
	repo := NewGameSetupRepository()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := repo.CreateGameSetup(ctx, testGameSetup("game-a", "player-a", "code-a")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create error = %v, want context.Canceled", err)
	}
	if _, err := repo.GetGameSetup(ctx, "game-a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get error = %v, want context.Canceled", err)
	}
	if _, err := repo.GetGameSetupByInviteCode(ctx, "code-a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetGameSetupByInviteCode error = %v, want context.Canceled", err)
	}
	if err := repo.SaveGameSetup(ctx, testGameSetup("game-a", "player-a", "code-a")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Save error = %v, want context.Canceled", err)
	}
	if err := repo.AddPlayerToGameSetup(ctx, "game-a", "player-b", 3); !errors.Is(err, context.Canceled) {
		t.Fatalf("AddPlayerToGameSetup error = %v, want context.Canceled", err)
	}
	if _, err := repo.ListGameSetupsForPlayer(ctx, "player-a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("List error = %v, want context.Canceled", err)
	}
}
