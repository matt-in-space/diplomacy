package lobby

import (
	"time"

	"github.com/matt-in-space/diplomacy/core/game"
)

type InviteStatus string

const (
	InvitePending  InviteStatus = "pending"
	InviteAccepted InviteStatus = "accepted"
	InviteDeclined InviteStatus = "declined"
)

// An Invite is one host's invitation for one email address to join a
// GameSetup. Code is the actual credential — whoever holds the link can
// accept, the same model as a Google Docs or Discord invite link — while
// Email is only a display label for the host's lobby view, not access
// control. This sidesteps needing real email delivery for v1 and avoids an
// email-mismatch failure mode (a host typo or an invitee with a different
// account email) that would otherwise strand the invite with no recovery
// path.
type Invite struct {
	Code        string // the credential — crypto/rand, same convention as session tokens
	GameID      game.GameID
	Email       string        // display label only, not access control
	PlayerID    game.PlayerID // "" until accepted
	Status      InviteStatus
	CreatedAt   time.Time
	RespondedAt time.Time
}
