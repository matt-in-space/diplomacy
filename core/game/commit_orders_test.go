package game_test

import (
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func TestGameCommitOrders(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)

	if err := g.CommitOrders("eng", gm); err != nil {
		t.Fatalf("CommitOrders failed: %v", err)
	}
	if _, ok := g.CommittedOrders["eng"]; !ok {
		t.Fatal("CommittedOrders does not contain eng")
	}
}

func TestGameCommitOrders_AcceptsRetreatsPhase(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	g.Turn.Phase = game.AcceptRetreats

	if err := g.CommitOrders("eng", gm); err != nil {
		t.Fatalf("CommitOrders failed: %v", err)
	}
	if _, ok := g.CommittedOrders["eng"]; !ok {
		t.Fatal("CommittedOrders does not contain eng")
	}
}

func TestGameCommitOrdersRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		edit func(g *game.Game, gm *gamemap.GameMap) (gamemap.NationID, *gamemap.GameMap)
		want string
	}{
		{
			name: "nil map",
			edit: func(g *game.Game, gm *gamemap.GameMap) (gamemap.NationID, *gamemap.GameMap) {
				return "eng", nil
			},
			want: "game map is required",
		},
		{
			name: "map mismatch",
			edit: func(g *game.Game, gm *gamemap.GameMap) (gamemap.NationID, *gamemap.GameMap) {
				return "eng", &gamemap.GameMap{ID: "other-map"}
			},
			want: "does not match",
		},
		{
			name: "wrong phase",
			edit: func(g *game.Game, gm *gamemap.GameMap) (gamemap.NationID, *gamemap.GameMap) {
				g.Turn.Phase = game.ResolveOrders
				return "eng", gm
			},
			want: "cannot commit orders during phase",
		},
		{
			name: "unknown nation",
			edit: func(g *game.Game, gm *gamemap.GameMap) (gamemap.NationID, *gamemap.GameMap) {
				return "ita", gm
			},
			want: "nation \"ita\" not found",
		},
		{
			name: "unassigned nation",
			edit: func(g *game.Game, gm *gamemap.GameMap) (gamemap.NationID, *gamemap.GameMap) {
				delete(g.Assignments, "fra")
				return "fra", gm
			},
			want: "nation \"fra\" is not assigned",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := loadWesternEuropeMap(t)
			g := newWesternEuropeGame(t, gm)
			nationID, commitMap := tt.edit(g, gm)

			err := g.CommitOrders(nationID, commitMap)
			if err == nil {
				t.Fatal("expected CommitOrders to fail")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("CommitOrders error = %q, want substring %q", err.Error(), tt.want)
			}
			if len(g.CommittedOrders) != 0 {
				t.Fatalf("CommittedOrders length = %d, want 0", len(g.CommittedOrders))
			}
		})
	}
}
