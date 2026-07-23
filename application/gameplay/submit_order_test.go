package gameplay

import (
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
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
			name:            "valid",
			gameID:          "test-game",
			playerID:        "player-a",
			expectedVersion: 1,
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
		{
			name:            "missing expected version",
			gameID:          "test-game",
			playerID:        "player-a",
			expectedVersion: 0,
			order:           validOrder,
			wantErr:         "expected version is required",
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
