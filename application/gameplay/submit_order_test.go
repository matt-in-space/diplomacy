package gameplay

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func TestNewSubmitOrderCommand(t *testing.T) {
	validOrder := game.NewHoldOrder("fra-army-par-start", "fra")

	tests := []struct {
		name            string
		gameID          game.GameID
		playerID        game.PlayerID
		expectedVersion uint64
		order           game.Order
		wantErr         string
	}{
		{
			name:            "valid with initial version",
			gameID:          "test-game",
			playerID:        "player-a",
			expectedVersion: 0,
			order:           validOrder,
		},
		{
			name:            "missing game ID",
			gameID:          "",
			playerID:        "player-a",
			expectedVersion: 1,
			order:           validOrder,
			wantErr:         "game ID is required",
		},
		{
			name:            "missing player ID",
			gameID:          "test-game",
			playerID:        "",
			expectedVersion: 1,
			order:           validOrder,
			wantErr:         "player ID is required",
		},
		{
			name:            "missing order",
			gameID:          "test-game",
			playerID:        "player-a",
			expectedVersion: 1,
			order:           nil,
			wantErr:         "order is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := NewSubmitOrderCommand(tt.gameID, tt.playerID, tt.expectedVersion, tt.order)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("NewSubmitOrderCommand() error = nil, want %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("NewSubmitOrderCommand() error = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewSubmitOrderCommand() unexpected error: %v", err)
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
			if cmd.Order != tt.order {
				t.Fatalf("cmd.Order = %+v, want %+v", cmd.Order, tt.order)
			}
		})
	}
}

func TestGameplayServiceSubmitOrder(t *testing.T) {
	g := repositoryTestGame("test-game")
	games := &submitOrderGameRepository{
		stored: StoredGame{Game: g, Version: 0},
	}
	maps := &submitOrderMapRepository{
		gameMap: submitOrderTestMap(),
	}
	service := NewGameplayService(games, nil, maps)
	order := game.NewHoldOrder("unit-a", "eng")
	cmd := SubmitOrderCommand{
		GameID:          g.ID,
		PlayerID:        "player-a",
		ExpectedVersion: 0,
		Order:           order,
	}

	if err := service.SubmitOrder(context.Background(), cmd); err != nil {
		t.Fatalf("SubmitOrder failed: %v", err)
	}
	if games.getGameID != cmd.GameID {
		t.Fatalf("GetGame game ID = %q, want %q", games.getGameID, cmd.GameID)
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
	if got := games.savedGame.Orders["unit-a"]; got != order {
		t.Fatalf("saved order = %+v, want %+v", got, order)
	}
}

func TestGameplayServiceSubmitOrderReturnsGameLookupError(t *testing.T) {
	lookupErr := errors.New("lookup failed")
	games := &submitOrderGameRepository{getErr: lookupErr}
	maps := &submitOrderMapRepository{gameMap: submitOrderTestMap()}
	service := NewGameplayService(games, nil, maps)

	err := service.SubmitOrder(context.Background(), submitOrderTestCommand())
	if !errors.Is(err, lookupErr) {
		t.Fatalf("SubmitOrder error = %v, want lookup error", err)
	}
	if maps.calls != 0 {
		t.Fatalf("GetMap calls = %d, want 0", maps.calls)
	}
	if games.saveCalls != 0 {
		t.Fatalf("SaveGame calls = %d, want 0", games.saveCalls)
	}
}

func TestGameplayServiceSubmitOrderRejectsUnauthorizedPlayer(t *testing.T) {
	games := &submitOrderGameRepository{
		stored: StoredGame{Game: repositoryTestGame("test-game"), Version: 0},
	}
	maps := &submitOrderMapRepository{gameMap: submitOrderTestMap()}
	service := NewGameplayService(games, nil, maps)
	cmd := submitOrderTestCommand()
	cmd.PlayerID = "other-player"

	err := service.SubmitOrder(context.Background(), cmd)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("SubmitOrder error = %v, want ErrUnauthorized", err)
	}
	if maps.calls != 0 {
		t.Fatalf("GetMap calls = %d, want 0", maps.calls)
	}
	if games.saveCalls != 0 {
		t.Fatalf("SaveGame calls = %d, want 0", games.saveCalls)
	}
}

func TestGameplayServiceSubmitOrderReturnsMapLookupError(t *testing.T) {
	mapErr := errors.New("map lookup failed")
	games := &submitOrderGameRepository{
		stored: StoredGame{Game: repositoryTestGame("test-game"), Version: 0},
	}
	maps := &submitOrderMapRepository{err: mapErr}
	service := NewGameplayService(games, nil, maps)

	err := service.SubmitOrder(context.Background(), submitOrderTestCommand())
	if !errors.Is(err, mapErr) {
		t.Fatalf("SubmitOrder error = %v, want map lookup error", err)
	}
	if games.saveCalls != 0 {
		t.Fatalf("SaveGame calls = %d, want 0", games.saveCalls)
	}
}

func TestGameplayServiceSubmitOrderReturnsOrderValidationError(t *testing.T) {
	g := repositoryTestGame("test-game")
	g.Turn.Phase = game.ResolveOrders
	games := &submitOrderGameRepository{
		stored: StoredGame{Game: g, Version: 0},
	}
	maps := &submitOrderMapRepository{gameMap: submitOrderTestMap()}
	service := NewGameplayService(games, nil, maps)

	err := service.SubmitOrder(context.Background(), submitOrderTestCommand())
	if err == nil || !strings.Contains(err.Error(), "failed to submit order") {
		t.Fatalf("SubmitOrder error = %v, want wrapped order validation error", err)
	}
	if games.saveCalls != 0 {
		t.Fatalf("SaveGame calls = %d, want 0", games.saveCalls)
	}
}

func TestGameplayServiceSubmitOrderReturnsSaveError(t *testing.T) {
	saveErr := errors.New("save failed")
	games := &submitOrderGameRepository{
		stored:  StoredGame{Game: repositoryTestGame("test-game"), Version: 0},
		saveErr: saveErr,
	}
	maps := &submitOrderMapRepository{gameMap: submitOrderTestMap()}
	service := NewGameplayService(games, nil, maps)

	err := service.SubmitOrder(context.Background(), submitOrderTestCommand())
	if !errors.Is(err, saveErr) {
		t.Fatalf("SubmitOrder error = %v, want save error", err)
	}
	if games.saveCalls != 1 {
		t.Fatalf("SaveGame calls = %d, want 1", games.saveCalls)
	}
}

func submitOrderTestCommand() SubmitOrderCommand {
	return SubmitOrderCommand{
		GameID:          "test-game",
		PlayerID:        "player-a",
		ExpectedVersion: 0,
		Order:           game.NewHoldOrder("unit-a", "eng"),
	}
}

func submitOrderTestMap() *gamemap.GameMap {
	return &gamemap.GameMap{
		ID:      "test-map",
		Nations: []gamemap.NationID{"eng"},
	}
}

type submitOrderGameRepository struct {
	stored               StoredGame
	getErr               error
	saveErr              error
	getGameID            game.GameID
	saveCalls            int
	savedGame            *game.Game
	savedExpectedVersion uint64
}

func (r *submitOrderGameRepository) CreateGame(context.Context, *game.Game) error {
	return nil
}

func (r *submitOrderGameRepository) GetGame(_ context.Context, gameID game.GameID) (StoredGame, error) {
	r.getGameID = gameID
	if r.getErr != nil {
		return StoredGame{}, r.getErr
	}
	return r.stored, nil
}

func (r *submitOrderGameRepository) SaveGame(_ context.Context, g *game.Game, expectedVersion uint64) (uint64, error) {
	r.saveCalls++
	r.savedGame = g
	r.savedExpectedVersion = expectedVersion
	if r.saveErr != nil {
		return 0, r.saveErr
	}
	return expectedVersion + 1, nil
}

type submitOrderMapRepository struct {
	gameMap  *gamemap.GameMap
	err      error
	calls    int
	getMapID gamemap.MapID
}

func (r *submitOrderMapRepository) GetMap(mapID gamemap.MapID) (*gamemap.GameMap, error) {
	r.calls++
	r.getMapID = mapID
	if r.err != nil {
		return nil, r.err
	}
	return r.gameMap, nil
}
