package gameplay

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func TestNewCommitOrdersCommand(t *testing.T) {
	tests := []struct {
		name            string
		gameID          game.GameID
		playerID        game.PlayerID
		expectedVersion uint64
		nationID        gamemap.NationID
		wantErr         string
	}{
		{
			name:            "valid with initial version",
			gameID:          "test-game",
			playerID:        "player-a",
			expectedVersion: 0,
			nationID:        "eng",
		},
		{
			name:            "missing game ID",
			playerID:        "player-a",
			expectedVersion: 1,
			nationID:        "eng",
			wantErr:         "game ID is required",
		},
		{
			name:            "missing player ID",
			gameID:          "test-game",
			expectedVersion: 1,
			nationID:        "eng",
			wantErr:         "player ID is required",
		},
		{
			name:            "missing nation ID",
			gameID:          "test-game",
			playerID:        "player-a",
			expectedVersion: 1,
			wantErr:         "nation ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := NewCommitOrdersCommand(tt.gameID, tt.playerID, tt.expectedVersion, tt.nationID)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("NewCommitOrdersCommand() error = nil, want %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("NewCommitOrdersCommand() error = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewCommitOrdersCommand() unexpected error: %v", err)
			}
			if cmd.GameID != tt.gameID {
				t.Fatalf("cmd.GameID = %q, want %q", cmd.GameID, tt.gameID)
			}
			if cmd.PlayerID != tt.playerID {
				t.Fatalf("cmd.PlayerID = %q, want %q", cmd.PlayerID, tt.playerID)
			}
			if cmd.ExpectedVersion != tt.expectedVersion {
				t.Fatalf("cmd.ExpectedVersion = %d, want %d", cmd.ExpectedVersion, tt.expectedVersion)
			}
			if cmd.NationID != tt.nationID {
				t.Fatalf("cmd.NationID = %q, want %q", cmd.NationID, tt.nationID)
			}
		})
	}
}

func TestGameplayServiceCommitOrders(t *testing.T) {
	g := repositoryTestGame("test-game")
	g.Assignments["fra"] = "player-b"
	games := &commitOrdersGameRepository{
		stored: StoredGame{Game: g, Version: 0},
	}
	maps := &commitOrdersMapRepository{gameMap: commitOrdersTestMap()}
	service := NewGameplayService(games, nil, maps)
	cmd := commitOrdersTestCommand()

	if err := service.CommitOrders(context.Background(), cmd); err != nil {
		t.Fatalf("CommitOrders failed: %v", err)
	}
	if games.getGameID != cmd.GameID {
		t.Fatalf("GetGame game ID = %q, want %q", games.getGameID, cmd.GameID)
	}
	if games.getCalls != 2 {
		t.Fatalf("GetGame calls = %d, want 2", games.getCalls)
	}
	if maps.getMapID != g.MapID {
		t.Fatalf("GetMap map ID = %q, want %q", maps.getMapID, g.MapID)
	}
	if games.saveCalls != 1 {
		t.Fatalf("SaveGame calls = %d, want 1", games.saveCalls)
	}
	if games.savedGame != g {
		t.Fatalf("SaveGame game = %p, want %p", games.savedGame, g)
	}
	if games.savedExpectedVersion != cmd.ExpectedVersion {
		t.Fatalf("SaveGame expected version = %d, want %d", games.savedExpectedVersion, cmd.ExpectedVersion)
	}
	if _, ok := games.savedGame.CommittedOrders[cmd.NationID]; !ok {
		t.Fatalf("saved game does not contain committed nation %q", cmd.NationID)
	}
}

func TestGameplayServiceCommitOrdersProcessesGameAfterFinalCommitment(t *testing.T) {
	ctx := context.Background()
	games := NewMemoryGameRepository()
	g := repositoryTestGame("test-game")
	if err := games.CreateGame(ctx, g); err != nil {
		t.Fatalf("CreateGame failed: %v", err)
	}
	maps := NewMemoryGameMapRepository(commitOrdersTestMap())
	service := NewGameplayService(games, nil, maps)

	if err := service.CommitOrders(ctx, commitOrdersTestCommand()); err != nil {
		t.Fatalf("CommitOrders failed: %v", err)
	}

	stored, err := games.GetGame(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetGame failed: %v", err)
	}
	// With no dislodged units, the accept retreats phase has no nations to
	// wait for and advances immediately, so processing runs one phase further
	// than it would with a dislodgement pending.
	if stored.Version != 4 {
		t.Fatalf("stored version = %d, want 4", stored.Version)
	}
	if stored.Game.Turn.Phase != game.ResolveRetreats {
		t.Fatalf("Turn.Phase = %q, want %q", stored.Game.Turn.Phase, game.ResolveRetreats)
	}
	if len(stored.Game.CommittedOrders) != 0 {
		t.Fatalf("CommittedOrders length = %d, want 0", len(stored.Game.CommittedOrders))
	}
}

func TestGameplayServiceCommitOrdersReturnsGameLookupError(t *testing.T) {
	lookupErr := errors.New("lookup failed")
	games := &commitOrdersGameRepository{getErr: lookupErr}
	maps := &commitOrdersMapRepository{gameMap: commitOrdersTestMap()}
	service := NewGameplayService(games, nil, maps)

	err := service.CommitOrders(context.Background(), commitOrdersTestCommand())
	if !errors.Is(err, lookupErr) {
		t.Fatalf("CommitOrders error = %v, want lookup error", err)
	}
	if maps.calls != 0 {
		t.Fatalf("GetMap calls = %d, want 0", maps.calls)
	}
	if games.saveCalls != 0 {
		t.Fatalf("SaveGame calls = %d, want 0", games.saveCalls)
	}
}

func TestGameplayServiceCommitOrdersRejectsUnauthorizedPlayer(t *testing.T) {
	games := &commitOrdersGameRepository{
		stored: StoredGame{Game: repositoryTestGame("test-game"), Version: 0},
	}
	maps := &commitOrdersMapRepository{gameMap: commitOrdersTestMap()}
	service := NewGameplayService(games, nil, maps)
	cmd := commitOrdersTestCommand()
	cmd.PlayerID = "other-player"

	err := service.CommitOrders(context.Background(), cmd)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("CommitOrders error = %v, want ErrUnauthorized", err)
	}
	if maps.calls != 0 {
		t.Fatalf("GetMap calls = %d, want 0", maps.calls)
	}
	if games.saveCalls != 0 {
		t.Fatalf("SaveGame calls = %d, want 0", games.saveCalls)
	}
}

func TestGameplayServiceCommitOrdersReturnsMapLookupError(t *testing.T) {
	mapErr := errors.New("map lookup failed")
	games := &commitOrdersGameRepository{
		stored: StoredGame{Game: repositoryTestGame("test-game"), Version: 0},
	}
	maps := &commitOrdersMapRepository{err: mapErr}
	service := NewGameplayService(games, nil, maps)

	err := service.CommitOrders(context.Background(), commitOrdersTestCommand())
	if !errors.Is(err, mapErr) {
		t.Fatalf("CommitOrders error = %v, want map lookup error", err)
	}
	if games.saveCalls != 0 {
		t.Fatalf("SaveGame calls = %d, want 0", games.saveCalls)
	}
}

func TestGameplayServiceCommitOrdersReturnsCommitError(t *testing.T) {
	g := repositoryTestGame("test-game")
	g.Assignments["ita"] = "player-a"
	games := &commitOrdersGameRepository{
		stored: StoredGame{Game: g, Version: 0},
	}
	maps := &commitOrdersMapRepository{gameMap: commitOrdersTestMap()}
	service := NewGameplayService(games, nil, maps)
	cmd := commitOrdersTestCommand()
	cmd.NationID = "ita"

	err := service.CommitOrders(context.Background(), cmd)
	if err == nil || !strings.Contains(err.Error(), "failed to commit orders") {
		t.Fatalf("CommitOrders error = %v, want wrapped commit error", err)
	}
	if games.saveCalls != 0 {
		t.Fatalf("SaveGame calls = %d, want 0", games.saveCalls)
	}
}

func TestGameplayServiceCommitOrdersReturnsSaveError(t *testing.T) {
	saveErr := errors.New("save failed")
	games := &commitOrdersGameRepository{
		stored:  StoredGame{Game: repositoryTestGame("test-game"), Version: 0},
		saveErr: saveErr,
	}
	maps := &commitOrdersMapRepository{gameMap: commitOrdersTestMap()}
	service := NewGameplayService(games, nil, maps)

	err := service.CommitOrders(context.Background(), commitOrdersTestCommand())
	if !errors.Is(err, saveErr) {
		t.Fatalf("CommitOrders error = %v, want save error", err)
	}
	if games.saveCalls != 1 {
		t.Fatalf("SaveGame calls = %d, want 1", games.saveCalls)
	}
}

func commitOrdersTestCommand() CommitOrdersCommand {
	return CommitOrdersCommand{
		GameID:          "test-game",
		PlayerID:        "player-a",
		ExpectedVersion: 0,
		NationID:        "eng",
	}
}

func commitOrdersTestMap() *gamemap.GameMap {
	return &gamemap.GameMap{
		ID:      "test-map",
		Nations: []gamemap.NationID{"eng"},
	}
}

type commitOrdersGameRepository struct {
	stored               StoredGame
	getErr               error
	saveErr              error
	getGameID            game.GameID
	getCalls             int
	saveCalls            int
	savedGame            *game.Game
	savedExpectedVersion uint64
}

func (r *commitOrdersGameRepository) CreateGame(context.Context, *game.Game) error {
	return nil
}

func (r *commitOrdersGameRepository) GetGame(_ context.Context, gameID game.GameID) (StoredGame, error) {
	r.getCalls++
	r.getGameID = gameID
	if r.getErr != nil {
		return StoredGame{}, r.getErr
	}
	return r.stored, nil
}

func (r *commitOrdersGameRepository) SaveGame(_ context.Context, g *game.Game, expectedVersion uint64) (uint64, error) {
	r.saveCalls++
	r.savedGame = g
	r.savedExpectedVersion = expectedVersion
	if r.saveErr != nil {
		return 0, r.saveErr
	}
	return expectedVersion + 1, nil
}

type commitOrdersMapRepository struct {
	gameMap  *gamemap.GameMap
	err      error
	calls    int
	getMapID gamemap.MapID
}

func (r *commitOrdersMapRepository) GetMap(mapID gamemap.MapID) (*gamemap.GameMap, error) {
	r.calls++
	r.getMapID = mapID
	if r.err != nil {
		return nil, r.err
	}
	return r.gameMap, nil
}
