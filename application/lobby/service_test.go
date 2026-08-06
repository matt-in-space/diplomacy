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
// stateful round-tripping (invite, accept, start, and check the resulting
// Game) a call-counting fake can't usefully stand in for. It also returns
// the underlying games and setups repositories directly, since Service has
// no read-only passthrough for either — tests that need to observe stored
// state beyond what a use-case method already returns go straight to the
// repository, the same way TestStartGameSucceeds... below checks the
// resulting Game via games.GetGame.
func newTestService(t *testing.T) (*lobby.Service, gameplay.GameRepository, lobby.GameSetupRepository) {
	t.Helper()
	setups := memory.NewGameSetupRepository()
	invites := memory.NewInviteRepository()
	games := memory.NewGameRepository()
	maps := memory.NewGameMapRepository(serviceTestMap())
	gp := gameplay.NewGameplayService(games, maps)
	return lobby.NewService(setups, invites, games, maps, gp), games, setups
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

func TestInvitePlayerDedupesRepeatInvite(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}

	first, err := service.InvitePlayer(ctx, setup.ID, "host-a", "a@example.com")
	if err != nil {
		t.Fatalf("first InvitePlayer failed: %v", err)
	}
	second, err := service.InvitePlayer(ctx, setup.ID, "host-a", "a@example.com")
	if err != nil {
		t.Fatalf("second InvitePlayer failed: %v", err)
	}
	if first.Code != second.Code {
		t.Fatalf("expected the same invite to be returned, got codes %q and %q", first.Code, second.Code)
	}
}

func TestInvitePlayerIssuesFreshInviteAfterDecline(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}

	first, err := service.InvitePlayer(ctx, setup.ID, "host-a", "a@example.com")
	if err != nil {
		t.Fatalf("first InvitePlayer failed: %v", err)
	}
	if err := service.DeclineInvite(ctx, first.Code, "player-a"); err != nil {
		t.Fatalf("DeclineInvite failed: %v", err)
	}

	second, err := service.InvitePlayer(ctx, setup.ID, "host-a", "a@example.com")
	if err != nil {
		t.Fatalf("second InvitePlayer failed: %v", err)
	}
	if second.Code == first.Code {
		t.Fatal("expected a fresh invite code after a decline")
	}
	if second.Status != lobby.InvitePending {
		t.Fatalf("second invite status = %q, want %q", second.Status, lobby.InvitePending)
	}
}

func TestInvitePlayerRejectsNonHost(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}

	_, err = service.InvitePlayer(ctx, setup.ID, "not-the-host", "a@example.com")
	if !errors.Is(err, lobby.ErrNotHost) {
		t.Fatalf("InvitePlayer error = %v, want ErrNotHost", err)
	}
}

func TestInvitePlayerRejectsInvalidEmail(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}

	if _, err := service.InvitePlayer(ctx, setup.ID, "host-a", "not-an-email"); err == nil {
		t.Fatal("expected InvitePlayer to reject an invalid email")
	}
}

func TestAcceptInviteRejectsSecondResponse(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}
	invite, err := service.InvitePlayer(ctx, setup.ID, "host-a", "a@example.com")
	if err != nil {
		t.Fatalf("InvitePlayer failed: %v", err)
	}

	if err := service.AcceptInvite(ctx, invite.Code, "player-a"); err != nil {
		t.Fatalf("AcceptInvite failed: %v", err)
	}
	err = service.AcceptInvite(ctx, invite.Code, "player-a")
	if !errors.Is(err, lobby.ErrInviteAlreadyResolved) {
		t.Fatalf("second AcceptInvite error = %v, want ErrInviteAlreadyResolved", err)
	}
	err = service.DeclineInvite(ctx, invite.Code, "player-a")
	if !errors.Is(err, lobby.ErrInviteAlreadyResolved) {
		t.Fatalf("DeclineInvite after accept error = %v, want ErrInviteAlreadyResolved", err)
	}
}

func TestStartGameSucceedsAndAssignsEveryAcceptedPlayer(t *testing.T) {
	service, games, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}

	invite, err := service.InvitePlayer(ctx, setup.ID, "host-a", "a@example.com")
	if err != nil {
		t.Fatalf("InvitePlayer failed: %v", err)
	}
	if err := service.AcceptInvite(ctx, invite.Code, "player-a"); err != nil {
		t.Fatalf("AcceptInvite failed: %v", err)
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

func TestStartGameRejectsRemainingPendingInvite(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	setup, err := service.CreateGameSetup(ctx, "host-a", "test-map")
	if err != nil {
		t.Fatalf("CreateGameSetup failed: %v", err)
	}
	if _, err := service.InvitePlayer(ctx, setup.ID, "host-a", "a@example.com"); err != nil {
		t.Fatalf("InvitePlayer failed: %v", err)
	}

	err = service.StartGame(ctx, setup.ID, "host-a")
	if !errors.Is(err, lobby.ErrInvitesPending) {
		t.Fatalf("StartGame error = %v, want ErrInvitesPending", err)
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
