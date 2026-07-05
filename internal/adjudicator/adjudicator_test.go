package adjudicator_test

import (
	"os"
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/internal/adjudicator"
	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

func TestResolve_ValidatesInputs(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) (*game.Game, *gamemap.GameMap)
		want  string
	}{
		{
			name: "nil game",
			setup: func(t *testing.T) (*game.Game, *gamemap.GameMap) {
				return nil, loadWesternEuropeMap(t)
			},
			want: "game is nil",
		},
		{
			name: "nil map",
			setup: func(t *testing.T) (*game.Game, *gamemap.GameMap) {
				g, _ := newResolutionGame(t)
				return g, nil
			},
			want: "map is nil",
		},
		{
			name: "map ID mismatch",
			setup: func(t *testing.T) (*game.Game, *gamemap.GameMap) {
				g, gm := newResolutionGame(t)
				gm.ID = "other-map"
				return g, gm
			},
			want: "map ID mismatch",
		},
		{
			name: "wrong phase",
			setup: func(t *testing.T) (*game.Game, *gamemap.GameMap) {
				gm := loadWesternEuropeMap(t)
				g := newGame(t, gm)
				return g, gm
			},
			want: "wrong turn phase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, gm := tt.setup(t)
			_, err := adjudicator.Resolve(g, gm)
			if err == nil {
				t.Fatalf("expected Resolve to fail")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Resolve error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestResolve_AcceptsValidInputs(t *testing.T) {
	g, gm := newResolutionGame(t)

	_, err := adjudicator.Resolve(g, gm)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
}

func newResolutionGame(t *testing.T) (*game.Game, *gamemap.GameMap) {
	t.Helper()

	gm := loadWesternEuropeMap(t)
	g := newGame(t, gm)
	g.Turn.Phase = game.ResolveOrders

	return g, gm
}

func newGame(t *testing.T, gm *gamemap.GameMap) *game.Game {
	t.Helper()

	g, err := game.NewGame(game.NewGameConfig{
		ID: "game-1",
		Assignments: map[gamemap.NationID]game.PlayerID{
			"eng": "player-1",
			"fra": "player-2",
		},
	}, gm)
	if err != nil {
		t.Fatalf("NewGame failed: %v", err)
	}

	return g
}

func loadWesternEuropeMap(t *testing.T) *gamemap.GameMap {
	t.Helper()

	data, err := os.ReadFile("../gamemap/testdata/western_europe.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	gm, err := gamemap.Load(data)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	return gm
}
