package lobby

import (
	"time"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// Status is a GameSetup's lifecycle state. It's never stored directly —
// Service.StatusFor computes it, since whether a setup is Active depends on
// whether a core/game.Game exists for it, and that's a fact a repository
// read has to answer, not something a struct field can hold without risking
// drifting out of sync with the real Game record.
type Status string

const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusCancelled Status = "cancelled"
)

// A GameSetup is a game in the recruiting/lobby phase, before a
// core/game.Game exists. It's a deliberately separate entity from Game
// rather than an extension of it — "still recruiting, nobody's confirmed
// yet" has no representation inside NewGame's assignment-driven model, and
// folding it in would leak a social/organizational concept into the pure
// rules engine.
type GameSetup struct {
	ID          game.GameID // becomes the real Game's ID once started — no remapping
	MapID       gamemap.MapID
	HostID      game.PlayerID
	CreatedAt   time.Time
	CancelledAt *time.Time // nil = not cancelled; the one fact that can't be derived
}
