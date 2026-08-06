package lobby_test

import (
	"context"
	"errors"
	"testing"

	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/application/lobby"
	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
	"github.com/matt-in-space/diplomacy/infrastructure/memory"
)

func serviceTestMap() *gamemap.GameMap {
	return &gamemap.GameMap{
		ID:      "test-map",
		Nations: []gamemap.NationID{"eng", "fra", "ger"},
	}
}

// newTestService wires a lobby.Service against real infrastructure/memory
// repositories, not fakes — this is exactly the kind of multi-step,
// stateful round-tripping (create, join, start, and check the resulting
// Game) a call-counting fake can't usefully stand in for. It also returns
// the underlying games and setups repositories directly, since Service has
// no read-only passthrough for either — tests that need to observe stored
// state beyond what a use-case method already returns go straight to the
// repository, the same way TestStartGameSucceeds... below checks the
// resulting Game via games.GetGame.
func newTestService(t *testing.T) (*lobby.Service, gameplay.GameRepository, lobby.GameSetupRepository) {
	t.Helper()
	setups := memory.NewGameSetupRepository()
	games := memory.NewGameRepository()
	maps := memory.NewGameMapRepository(serviceTestMap())
	gp := gameplay.NewGameplayService(games, maps)
	return lobby.NewService(setups, games, maps, gp), games, setups
}

func TestCreateGameSetup(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()

	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}
	if setup.HostID != "host-a" {
		t.Fatalf("HostID = %q, want %q", setup.HostID, "host-a")
	}
	if setup.InviteCode == "" {
		t.Fatal("expected a non-empty InviteCode")
	}
	if len(setup.PlayerIDs) != 1 || setup.PlayerIDs[0] != "host-a" {
		t.Fatalf("PlayerIDs = %v, want [host-a]", setup.PlayerIDs)
	}

	status, err := service.StatusFor(ctx, setup)
	if err != nil {
		t.Fatalf("StatusFor failed: %v", err)
	}
	if status != lobby.StatusPending {
		t.Fatalf("status = %q, want %q", status, lobby.StatusPending)
	}
}

func TestCreateGameSetupRejectsUnknownMap(t *testing.T) {
	service, _, _ := newTestService(t)

	_, err := service.CreateGameSetup(context.Background(), "host-a", "missing-map")
	if !errors.Is(err, gameplay.ErrMapNotFound) {
		t.Fatalf("CreateGameSetup error = %v, want ErrMapNotFound", err)
	}
}

func TestJoinGameSetupAddsPlayer(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}

	updated, err := service.JoinGameSetup(ctx, setup.InviteCode, "player-a")
	if err != nil {
		t.Fatalf("JoinGameSetup failed: %v", err)
	}
	if len(updated.PlayerIDs) != 2 {
		t.Fatalf("PlayerIDs = %v, want 2 entries", updated.PlayerIDs)
	}
}

func TestJoinGameSetupIsIdempotent(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}

	if _, err := service.JoinGameSetup(ctx, setup.InviteCode, "player-a"); err != nil {
		t.Fatalf("first JoinGameSetup failed: %v", err)
	}
	updated, err := service.JoinGameSetup(ctx, setup.InviteCode, "player-a")
	if err != nil {
		t.Fatalf("second JoinGameSetup failed: %v", err)
	}
	if len(updated.PlayerIDs) != 2 {
		t.Fatalf("PlayerIDs = %v, want 2 entries after rejoining", updated.PlayerIDs)
	}
}

func TestJoinGameSetupHostUsingOwnCodeIsNoOp(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}

	updated, err := service.JoinGameSetup(ctx, setup.InviteCode, "host-a")
	if err != nil {
		t.Fatalf("JoinGameSetup(host) failed: %v", err)
	}
	if len(updated.PlayerIDs) != 1 {
		t.Fatalf("PlayerIDs = %v, want [host-a] unchanged", updated.PlayerIDs)
	}
}

func TestJoinGameSetupRejectsUnknownCode(t *testing.T) {
	service, _, _ := newTestService(t)

	_, err := service.JoinGameSetup(context.Background(), "missing-code", "player-a")
	if !errors.Is(err, lobby.ErrGameSetupNotFound) {
		t.Fatalf("JoinGameSetup error = %v, want ErrGameSetupNotFound", err)
	}
}

func TestJoinGameSetupRejectsCancelledSetup(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}
	if err := service.CancelGameSetup(ctx, setup.ID, "host-a"); err != nil {
		t.Fatalf("CancelGameSetup failed: %v", err)
	}

	_, err = service.JoinGameSetup(ctx, setup.InviteCode, "player-a")
	if !errors.Is(err, lobby.ErrGameSetupNotOpen) {
		t.Fatalf("JoinGameSetup error = %v, want ErrGameSetupNotOpen", err)
	}
}

func TestJoinGameSetupRejectsStartedSetup(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}
	if err := service.StartGame(ctx, setup.ID, "host-a"); err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	_, err = service.JoinGameSetup(ctx, setup.InviteCode, "player-a")
	if !errors.Is(err, lobby.ErrGameSetupNotOpen) {
		t.Fatalf("JoinGameSetup error = %v, want ErrGameSetupNotOpen", err)
	}
}

func TestJoinGameSetupRejectsPastCapacity(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	// serviceTestMap has 3 nations: host-a + 2 joiners fills it exactly.
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}
	if _, err := service.JoinGameSetup(ctx, setup.InviteCode, "player-a"); err != nil {
		t.Fatalf("JoinGameSetup(player-a) failed: %v", err)
	}
	if _, err := service.JoinGameSetup(ctx, setup.InviteCode, "player-b"); err != nil {
		t.Fatalf("JoinGameSetup(player-b) failed: %v", err)
	}

	_, err = service.JoinGameSetup(ctx, setup.InviteCode, "player-c")
	if !errors.Is(err, lobby.ErrGameSetupFull) {
		t.Fatalf("JoinGameSetup(player-c) error = %v, want ErrGameSetupFull", err)
	}
}

func TestStartGameSucceedsAndAssignsEveryPlayer(t *testing.T) {
	service, games, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}
	if _, err := service.JoinGameSetup(ctx, setup.InviteCode, "player-a"); err != nil {
		t.Fatalf("JoinGameSetup failed: %v", err)
	}

	if err := service.StartGame(ctx, setup.ID, "host-a"); err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	status, err := service.StatusFor(ctx, setup)
	if err != nil {
		t.Fatalf("StatusFor failed: %v", err)
	}
	if status != lobby.StatusActive {
		t.Fatalf("status = %q, want %q", status, lobby.StatusActive)
	}

	stored, err := games.GetGame(ctx, setup.ID)
	if err != nil {
		t.Fatalf("GetGame failed: %v", err)
	}

	assignedPlayers := make(map[game.PlayerID]bool)
	for _, playerID := range stored.Game.Assignments {
		assignedPlayers[playerID] = true
	}
	if !assignedPlayers["host-a"] {
		t.Fatal("expected host-a to be assigned a nation")
	}
	if !assignedPlayers["player-a"] {
		t.Fatal("expected player-a to be assigned a nation")
	}
	if len(stored.Game.Assignments) != 2 {
		t.Fatalf("len(Assignments) = %d, want 2", len(stored.Game.Assignments))
	}
}

func TestStartGameRejectsNonHost(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}

	err = service.StartGame(ctx, setup.ID, "not-the-host")
	if !errors.Is(err, lobby.ErrNotHost) {
		t.Fatalf("StartGame error = %v, want ErrNotHost", err)
	}
}

func TestStartGameRejectsDoubleStart(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}
	if err := service.StartGame(ctx, setup.ID, "host-a"); err != nil {
		t.Fatalf("first StartGame failed: %v", err)
	}

	err = service.StartGame(ctx, setup.ID, "host-a")
	if !errors.Is(err, lobby.ErrGameSetupNotOpen) {
		t.Fatalf("second StartGame error = %v, want ErrGameSetupNotOpen", err)
	}
}

func TestCancelGameSetupIsIdempotent(t *testing.T) {
	service, _, setups := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}

	if err := service.CancelGameSetup(ctx, setup.ID, "host-a"); err != nil {
		t.Fatalf("first CancelGameSetup failed: %v", err)
	}
	if err := service.CancelGameSetup(ctx, setup.ID, "host-a"); err != nil {
		t.Fatalf("second CancelGameSetup failed: %v", err)
	}

	// StatusFor reads CancelledAt off the struct it's given, not by
	// re-fetching from the repository — so this must pass the
	// post-cancellation snapshot, not the stale one CreateGameSetup
	// returned.
	setup, err = setups.GetGameSetup(ctx, setup.ID)
	if err != nil {
		t.Fatalf("GetGameSetup failed: %v", err)
	}
	status, err := service.StatusFor(ctx, setup)
	if err != nil {
		t.Fatalf("StatusFor failed: %v", err)
	}
	if status != lobby.StatusCancelled {
		t.Fatalf("status = %q, want %q", status, lobby.StatusCancelled)
	}
}

func TestCancelGameSetupRejectsNonHost(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}

	err = service.CancelGameSetup(ctx, setup.ID, "not-the-host")
	if !errors.Is(err, lobby.ErrNotHost) {
		t.Fatalf("CancelGameSetup error = %v, want ErrNotHost", err)
	}
}

func TestCancelGameSetupRejectsAlreadyStarted(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}
	if err := service.StartGame(ctx, setup.ID, "host-a"); err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	err = service.CancelGameSetup(ctx, setup.ID, "host-a")
	if !errors.Is(err, lobby.ErrGameSetupNotOpen) {
		t.Fatalf("CancelGameSetup error = %v, want ErrGameSetupNotOpen", err)
	}
}
