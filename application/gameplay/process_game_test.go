package gameplay

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func TestGameplayServiceProcessGameWaitsForCommittedOrders(t *testing.T) {
	games := &processGameRepository{
		stored: StoredGame{Game: repositoryTestGame("test-game"), Version: 3},
	}
	maps := &processGameMapRepository{gameMap: processGameTestMap()}
	service := NewGameplayService(games, nil, maps)

	if err := service.ProcessGame(context.Background(), "test-game"); err != nil {
		t.Fatalf("ProcessGame failed: %v", err)
	}
	if games.getGameID != "test-game" {
		t.Fatalf("GetGame game ID = %q, want test-game", games.getGameID)
	}
	if games.saveCalls != 0 {
		t.Fatalf("SaveGame calls = %d, want 0", games.saveCalls)
	}
	if maps.calls != 0 {
		t.Fatalf("GetMap calls = %d, want 0", maps.calls)
	}
}

func TestGameplayServiceProcessGameProcessesReadyOrders(t *testing.T) {
	g := repositoryTestGame("test-game")
	g.CommittedOrders["eng"] = struct{}{}
	g.Orders["unit-a"] = game.NewHoldOrder("unit-a", "eng")
	games := &processGameRepository{
		stored: StoredGame{Game: g, Version: 3},
	}
	maps := &processGameMapRepository{gameMap: processGameTestMap()}
	service := NewGameplayService(games, nil, maps)

	if err := service.ProcessGame(context.Background(), g.ID); err != nil {
		t.Fatalf("ProcessGame failed: %v", err)
	}
	// repositoryTestGame's single unit is never dislodged, so accept retreats
	// has no nations to wait for and resolve retreats has nothing to decide:
	// four phases run (resolve orders, accept retreats, resolve retreats,
	// and into fall's accept orders) before the loop stops waiting for a
	// commitment that hasn't happened yet.
	if games.saveCalls != 4 {
		t.Fatalf("SaveGame calls = %d, want 4", games.saveCalls)
	}
	if got, want := games.expectedVersions, []uint64{3, 4, 5, 6}; !equalVersions(got, want) {
		t.Fatalf("SaveGame expected versions = %v, want %v", got, want)
	}
	if got, want := games.savedPhases, []game.Phase{game.ResolveOrders, game.AcceptRetreats, game.ResolveRetreats, game.AcceptOrders}; !equalPhases(got, want) {
		t.Fatalf("saved phases = %v, want %v", got, want)
	}
	if maps.calls != 2 {
		t.Fatalf("GetMap calls = %d, want 2", maps.calls)
	}
	if g.Turn.Phase != game.AcceptOrders {
		t.Fatalf("Turn.Phase = %q, want %q", g.Turn.Phase, game.AcceptOrders)
	}
	if g.Turn.Season != game.Fall {
		t.Fatalf("Turn.Season = %q, want %q", g.Turn.Season, game.Fall)
	}
	if len(g.CommittedOrders) != 0 {
		t.Fatalf("CommittedOrders length = %d, want 0", len(g.CommittedOrders))
	}
	if len(g.Orders) != 0 {
		t.Fatalf("Orders length = %d, want 0", len(g.Orders))
	}
}

func TestGameplayServiceProcessGameProcessesResolutionPhase(t *testing.T) {
	g := repositoryTestGame("test-game")
	g.Turn.Phase = game.ResolveOrders
	games := &processGameRepository{
		stored: StoredGame{Game: g, Version: 6},
	}
	maps := &processGameMapRepository{gameMap: processGameTestMap()}
	service := NewGameplayService(games, nil, maps)

	if err := service.ProcessGame(context.Background(), g.ID); err != nil {
		t.Fatalf("ProcessGame failed: %v", err)
	}
	// As above, no dislodged units means accept retreats and resolve retreats
	// both run without anyone needing to commit.
	if games.saveCalls != 3 {
		t.Fatalf("SaveGame calls = %d, want 3", games.saveCalls)
	}
	if games.expectedVersions[0] != 6 {
		t.Fatalf("SaveGame expected version = %d, want 6", games.expectedVersions[0])
	}
	if g.Turn.Phase != game.AcceptOrders {
		t.Fatalf("Turn.Phase = %q, want %q", g.Turn.Phase, game.AcceptOrders)
	}
}

func TestGameplayServiceProcessGameReturnsGameLookupError(t *testing.T) {
	lookupErr := errors.New("lookup failed")
	games := &processGameRepository{getErr: lookupErr}
	maps := &processGameMapRepository{gameMap: processGameTestMap()}
	service := NewGameplayService(games, nil, maps)

	err := service.ProcessGame(context.Background(), "test-game")
	if !errors.Is(err, lookupErr) {
		t.Fatalf("ProcessGame error = %v, want lookup error", err)
	}
	if maps.calls != 0 {
		t.Fatalf("GetMap calls = %d, want 0", maps.calls)
	}
	if games.saveCalls != 0 {
		t.Fatalf("SaveGame calls = %d, want 0", games.saveCalls)
	}
}

func TestGameplayServiceProcessGameReturnsMapLookupError(t *testing.T) {
	mapErr := errors.New("map lookup failed")
	g := repositoryTestGame("test-game")
	g.Turn.Phase = game.ResolveOrders
	games := &processGameRepository{
		stored: StoredGame{Game: g, Version: 0},
	}
	maps := &processGameMapRepository{err: mapErr}
	service := NewGameplayService(games, nil, maps)

	err := service.ProcessGame(context.Background(), g.ID)
	if !errors.Is(err, mapErr) {
		t.Fatalf("ProcessGame error = %v, want map lookup error", err)
	}
	if games.saveCalls != 0 {
		t.Fatalf("SaveGame calls = %d, want 0", games.saveCalls)
	}
}

func TestGameplayServiceProcessGameReturnsResolutionError(t *testing.T) {
	g := repositoryTestGame("test-game")
	g.Turn.Phase = game.ResolveOrders
	g.CommittedOrders["eng"] = struct{}{}
	g.Orders["unit-a"] = game.NewHoldOrder("unit-a", "eng")
	games := &processGameRepository{
		stored: StoredGame{Game: g, Version: 0},
	}
	maps := &processGameMapRepository{
		gameMap: &gamemap.GameMap{ID: "other-map", Nations: []gamemap.NationID{"eng"}},
	}
	service := NewGameplayService(games, nil, maps)

	err := service.ProcessGame(context.Background(), g.ID)
	if err == nil || !strings.Contains(err.Error(), "failed to resolve game") {
		t.Fatalf("ProcessGame error = %v, want wrapped resolution error", err)
	}
	if games.saveCalls != 0 {
		t.Fatalf("SaveGame calls = %d, want 0", games.saveCalls)
	}
	if g.Turn.Phase != game.ResolveOrders {
		t.Fatalf("Turn.Phase = %q, want %q", g.Turn.Phase, game.ResolveOrders)
	}
	if len(g.CommittedOrders) != 1 {
		t.Fatalf("CommittedOrders length = %d, want 1", len(g.CommittedOrders))
	}
	if len(g.Orders) != 1 {
		t.Fatalf("Orders length = %d, want 1", len(g.Orders))
	}
}

func TestGameplayServiceProcessGameReturnsSaveError(t *testing.T) {
	saveErr := errors.New("save failed")
	g := repositoryTestGame("test-game")
	g.CommittedOrders["eng"] = struct{}{}
	games := &processGameRepository{
		stored:  StoredGame{Game: g, Version: 2},
		saveErr: saveErr,
	}
	maps := &processGameMapRepository{gameMap: processGameTestMap()}
	service := NewGameplayService(games, nil, maps)

	err := service.ProcessGame(context.Background(), g.ID)
	if !errors.Is(err, saveErr) {
		t.Fatalf("ProcessGame error = %v, want save error", err)
	}
	if games.saveCalls != 1 {
		t.Fatalf("SaveGame calls = %d, want 1", games.saveCalls)
	}
	if maps.calls != 0 {
		t.Fatalf("GetMap calls = %d, want 0", maps.calls)
	}
}

func TestGameplayServiceProcessGameIgnoresUnsupportedPhase(t *testing.T) {
	g := repositoryTestGame("test-game")
	g.Turn.Phase = game.AcceptAdjustments
	games := &processGameRepository{
		stored: StoredGame{Game: g, Version: 0},
	}
	maps := &processGameMapRepository{gameMap: processGameTestMap()}
	service := NewGameplayService(games, nil, maps)

	if err := service.ProcessGame(context.Background(), g.ID); err != nil {
		t.Fatalf("ProcessGame failed: %v", err)
	}
	if games.saveCalls != 0 {
		t.Fatalf("SaveGame calls = %d, want 0", games.saveCalls)
	}
	if maps.calls != 0 {
		t.Fatalf("GetMap calls = %d, want 0", maps.calls)
	}
}

func TestGameplayServiceProcessGameSkipsRetreatsWithNoDislodgements(t *testing.T) {
	g := repositoryTestGame("test-game")
	g.Turn.Phase = game.AcceptRetreats
	games := &processGameRepository{
		stored: StoredGame{Game: g, Version: 0},
	}
	maps := &processGameMapRepository{gameMap: processGameTestMap()}
	service := NewGameplayService(games, nil, maps)

	if err := service.ProcessGame(context.Background(), g.ID); err != nil {
		t.Fatalf("ProcessGame failed: %v", err)
	}
	// No unit is dislodged, so there is no nation to wait for a commitment
	// from, and resolve retreats has nothing to decide either: both phases
	// run back-to-back.
	if games.saveCalls != 2 {
		t.Fatalf("SaveGame calls = %d, want 2", games.saveCalls)
	}
	if maps.calls != 1 {
		t.Fatalf("GetMap calls = %d, want 1", maps.calls)
	}
	if g.Turn.Phase != game.AcceptOrders {
		t.Fatalf("Turn.Phase = %q, want %q", g.Turn.Phase, game.AcceptOrders)
	}
}

func TestGameplayServiceProcessGameWaitsForRetreatCommitment(t *testing.T) {
	g := repositoryTestGame("test-game")
	g.Turn.Phase = game.AcceptRetreats
	dislodgeUnit(t, g, "unit-a")
	games := &processGameRepository{
		stored: StoredGame{Game: g, Version: 0},
	}
	maps := &processGameMapRepository{gameMap: processGameTestMap()}
	service := NewGameplayService(games, nil, maps)

	if err := service.ProcessGame(context.Background(), g.ID); err != nil {
		t.Fatalf("ProcessGame failed: %v", err)
	}
	if games.saveCalls != 0 {
		t.Fatalf("SaveGame calls = %d, want 0", games.saveCalls)
	}
	if g.Turn.Phase != game.AcceptRetreats {
		t.Fatalf("Turn.Phase = %q, want %q", g.Turn.Phase, game.AcceptRetreats)
	}
}

func TestGameplayServiceProcessGameProcessesRetreatCommitment(t *testing.T) {
	g := repositoryTestGame("test-game")
	g.Turn.Phase = game.AcceptRetreats
	dislodgeUnit(t, g, "unit-a")
	g.CommittedOrders["eng"] = struct{}{}
	games := &processGameRepository{
		stored: StoredGame{Game: g, Version: 0},
	}
	maps := &processGameMapRepository{gameMap: processGameTestMap()}
	service := NewGameplayService(games, nil, maps)

	if err := service.ProcessGame(context.Background(), g.ID); err != nil {
		t.Fatalf("ProcessGame failed: %v", err)
	}
	if games.saveCalls != 2 {
		t.Fatalf("SaveGame calls = %d, want 2", games.saveCalls)
	}
	if g.Turn.Phase != game.AcceptOrders {
		t.Fatalf("Turn.Phase = %q, want %q", g.Turn.Phase, game.AcceptOrders)
	}
	// No retreat order was submitted for the dislodged unit, so it force-
	// disbands rather than retreating.
	if _, ok := g.Units["unit-a"]; ok {
		t.Fatal("unit-a should have been disbanded, but is still present")
	}
}

func processGameTestMap() *gamemap.GameMap {
	return &gamemap.GameMap{
		ID:      "test-map",
		Nations: []gamemap.NationID{"eng"},
	}
}

type processGameRepository struct {
	stored           StoredGame
	getErr           error
	saveErr          error
	getGameID        game.GameID
	saveCalls        int
	expectedVersions []uint64
	savedPhases      []game.Phase
}

func (r *processGameRepository) CreateGame(context.Context, *game.Game) error {
	return nil
}

func (r *processGameRepository) GetGame(_ context.Context, gameID game.GameID) (StoredGame, error) {
	r.getGameID = gameID
	if r.getErr != nil {
		return StoredGame{}, r.getErr
	}
	return r.stored, nil
}

func (r *processGameRepository) SaveGame(_ context.Context, g *game.Game, expectedVersion uint64) (uint64, error) {
	r.saveCalls++
	r.expectedVersions = append(r.expectedVersions, expectedVersion)
	r.savedPhases = append(r.savedPhases, g.Turn.Phase)
	if r.saveErr != nil {
		return 0, r.saveErr
	}
	return expectedVersion + 1, nil
}

type processGameMapRepository struct {
	gameMap *gamemap.GameMap
	err     error
	calls   int
}

func (r *processGameMapRepository) GetMap(gamemap.MapID) (*gamemap.GameMap, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	return r.gameMap, nil
}

func equalVersions(got, want []uint64) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func equalPhases(got, want []game.Phase) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
