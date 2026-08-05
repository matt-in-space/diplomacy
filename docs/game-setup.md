# Game Setup Flow

## Purpose

This document captures the design for getting from "I want to play a game
with some friends" to an actual `core/game.Game` that's ready for orders —
listing your games, creating one, inviting people, accepting/declining, and
kicking off. Same spirit as `docs/user-experience.md`: write decisions down
as they're made, flag what's still open, let it evolve as it's actually
built. Nothing here is implemented yet.

This is squarely account/lobby territory per the two-flow split in
`docs/user-experience.md` — Go-rendered pages, not the future Svelte SPA.
The actual game screen, once a setup goes active, is explicitly out of
scope here.

## Why this isn't just `StoredGame`

`StoredGame` wraps `core/game.Game`, and `game.NewGame` immediately seeds
`Units` from the map and expects an assignment map (which can be partial —
`NewGame` never required full nation coverage, verified against the current
code). There's no way to represent "still recruiting players, nobody's
confirmed yet" inside it without leaking a social/organizational concept
into the pure rules engine — the same reasoning that kept `auth.Player`'s
credentials out of `core/game.Player`. This needs a genuinely separate
entity, `GameSetup`, that exists *before* a `core/game.Game` does and
produces one once it's ready.

Invite status lives in its own `Invite` records, not nested inside
`GameSetup` — matching the "one repository per resource" principle already
applied to `Player`/`Session`, and matching the relational split already
sketched in `docs/user-experience.md`'s database section: `games` metadata
and `invites`/`assignments` as separate tables.

## Decisions

- **Invite mechanism: a link, not a sent email.** An `Invite` stores both an
  email (a label — who the host meant this for, shown in their lobby view)
  and a random `Code` (the actual credential). Whoever holds the link can
  accept — same model as a Google Docs or Discord invite link, no
  email-matching failure mode to design around (a host typo or an invitee
  with a different account email would otherwise strand the invite with no
  recovery path). This sidesteps needing real email delivery for v1 — no
  provider is chosen yet, Mailpit is dev-only — while keeping the email on
  record for whenever real delivery exists later. `docs/user-experience.md`
  already anticipated this: *"an invite link needs to open cold... without
  hydrating a JS bundle first."*
- **Starting: happy-path gating.** "Start Game" becomes available once
  there are zero *pending* invites left (everyone's responded, accept or
  decline) — no timeouts, no auto-expiry, no reminders. A decline just sits
  there; the host sends a new invite to fill the slot. No minimum player
  count beyond "the host is always in" — a host judgment call, not a
  validated rule, for v1.
- **Nation assignment: random shuffle at kickoff.** Accepted players
  (including the host) are randomly assigned to the map's nations; fewer
  accepted players than nations just leaves the remainder vacant (already
  supported). Host-controlled or player-picked assignment is a reasonable
  later feature, not a redesign to retrofit.
- **Kickoff "notifications" = visibility, not delivery.** For v1, starting
  a game just needs to make it show up on the players' `/games` list — no
  real email or push. Real-time "your game just started" notification is a
  natural fit for the WebSocket channel already planned for the in-game
  SPA, not something to build now.
- **The host is a participant by construction**, not via an invite record —
  they don't invite themselves; `GameSetup.HostID` counts as accepted for
  start-readiness.
- **Cancelling a pending setup** is in scope alongside starting (host-only,
  small) — cheap enough that a stalled lobby with unresponsive invitees
  isn't permanently stuck with no way out.

## Design

### `application/lobby` (new package)

```go
type Status string
const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusCancelled Status = "cancelled"
)

type GameSetup struct {
	ID        game.GameID // becomes the real Game's ID once started — no remapping
	MapID     gamemap.MapID
	HostID    game.PlayerID
	Status    Status
	CreatedAt time.Time
}

type InviteStatus string
const (
	InvitePending  InviteStatus = "pending"
	InviteAccepted InviteStatus = "accepted"
	InviteDeclined InviteStatus = "declined"
)

type Invite struct {
	Code        string // the credential — crypto/rand, same convention as session tokens
	GameID      game.GameID
	Email       string // display label only, not access control
	PlayerID    game.PlayerID // "" until accepted
	Status      InviteStatus
	CreatedAt   time.Time
	RespondedAt time.Time
}
```

Two repositories (interfaces here, in-memory implementations in
`infrastructure/memory`, same split as every other resource):

```go
type GameSetupRepository interface {
	CreateGameSetup(ctx context.Context, setup *GameSetup) error
	GetGameSetup(ctx context.Context, id game.GameID) (*GameSetup, error)
	SaveGameSetup(ctx context.Context, setup *GameSetup) error
	ListGameSetupsForPlayer(ctx context.Context, playerID game.PlayerID) ([]*GameSetup, error)
}

type InviteRepository interface {
	CreateInvite(ctx context.Context, invite *Invite) error
	GetInviteByCode(ctx context.Context, code string) (*Invite, error)
	SaveInvite(ctx context.Context, invite *Invite) error
	ListInvitesForGame(ctx context.Context, gameID game.GameID) ([]*Invite, error)
	ListInvitesForEmail(ctx context.Context, email string) ([]*Invite, error) // pending invites addressed to a logged-in player, independent of having the link
}
```

`Service`, mirroring `auth.Service`'s shape:

```go
type Service struct {
	setups   GameSetupRepository
	invites  InviteRepository
	players  auth.PlayerRepository
	gameplay *gameplay.GameplayService
	maps     gameplay.GameMapRepository
}

func (s *Service) CreateGameSetup(ctx context.Context, hostID game.PlayerID, mapID gamemap.MapID) (*GameSetup, error)
func (s *Service) InvitePlayer(ctx context.Context, gameID game.GameID, requesterID game.PlayerID, email string) (*Invite, error) // host-only, dedupes an existing invite for the same email
func (s *Service) AcceptInvite(ctx context.Context, code string, playerID game.PlayerID) error
func (s *Service) DeclineInvite(ctx context.Context, code string, playerID game.PlayerID) error
func (s *Service) StartGame(ctx context.Context, gameID game.GameID, requesterID game.PlayerID) error // host-only; gates on zero pending invites; random nation shuffle; calls gameplay.CreateGame; flips setup to Active
func (s *Service) CancelGameSetup(ctx context.Context, gameID game.GameID, requesterID game.PlayerID) error
```

`StartGame`'s core logic:

```go
accepted := acceptedPlayerIDs(invites) // + HostID
nations := slices.Clone(gm.Nations)
rand.Shuffle(len(nations), ...)
rand.Shuffle(len(accepted), ...)
assignments := map[gamemap.NationID]game.PlayerID{}
for i, playerID := range accepted {
	if i >= len(nations) { break } // more accepted than nations — host over-invited; extras simply don't get assigned
	assignments[nations[i]] = playerID
}
_, err := s.gameplay.CreateGame(ctx, setup.ID, setup.MapID, assignments)
```

### `application/gameplay` — new use cases

Nothing has ever actually *created* a game before now — only tests called
`GameRepository.CreateGame` directly. This is where that becomes a real use
case, the same shape as every other `GameplayService` method:

```go
// create_game.go
func (s *GameplayService) CreateGame(ctx context.Context, id game.GameID, mapID gamemap.MapID, assignments map[gamemap.NationID]game.PlayerID) (StoredGame, error)
// list_games.go
func (s *GameplayService) ListGamesForPlayer(ctx context.Context, playerID game.PlayerID) ([]StoredGame, error)
```

`lobby.Service` calls *into* `GameplayService.CreateGame` at kickoff rather
than constructing games itself — `lobby` decides who/when, `gameplay` still
owns how a game actually gets created and persisted.

`GameRepository` gains `ListGamesForPlayer` — its first new method since it
was built. Nothing today can answer "which games is this player in"; every
existing method requires already knowing the game ID. In-memory: a plain
linear scan over stored games' `Assignments`. Postgres will eventually want
a real indexed query against the `games` metadata table instead of scanning
JSONB — already the documented plan, not a new decision.

`cmd/server/main.go`'s `GameplayService` stops being discarded at this
point — the same milestone `pr`/`sr` hit in auth Part 1.

### `web` — new routes

```
GET  /games                    require login — lists: active games (GameplayService.ListGamesForPlayer),
                                pending setups you host or belong to (GameSetupRepository.ListGameSetupsForPlayer),
                                pending invites addressed to your email (InviteRepository.ListInvitesForEmail)
GET  /games/new                require login — create-game form (map choice trivial: one option today)
POST /games                    creates an empty GameSetup, redirects to its lobby page
GET  /games/{id}               require login — pending: lobby view (invite list + invite form + start/cancel, host-only actions);
                                active: minimal stub ("game started" — the real game screen is future work)
POST /games/{id}/invites       host-only — add an invite by email
POST /games/{id}/start         host-only — gated on zero pending invites
POST /games/{id}/cancel        host-only
GET  /invites/{code}           require login (redirect through /login?next=... if not) — accept/decline landing page
POST /invites/{code}/accept
POST /invites/{code}/decline
```

New middleware, alongside `web/session_middleware.go`'s existing
`withCurrentPlayer`:

```go
// requireAuthentication redirects to /login?next=<this URL> if there's no
// current player. The counterpart to withCurrentPlayer's non-blocking
// style — the first routes in the app that actually need one.
func requireAuthentication(next http.Handler) http.Handler
```

`handleLoginForm` renders the `next` query param into a hidden form field;
`handleLoginSubmit` redirects there instead of `/` on success — but only if
it's a validated same-site relative path (starts with exactly one `/`,
never `//` or a full URL), to avoid becoming an open redirect.

New templates, through the existing `parsePage("templates/<name>.html")` +
shared-layout pattern: `games.html` (the list), `games_new.html`,
`game_setup.html` (lobby detail), `invite.html` (accept/decline landing).

## Suggested build order (not decided — flagging the option)

Big enough to consider splitting the same way auth was: **backend first**
(`application/lobby`, the new `gameplay`/`GameRepository` methods,
in-memory repos, tests — verified via direct service tests and `httptest`,
no pages) — **then** the web pages on top. Not committed to; just noting
the option was there for auth and worked well.

## Out of scope

- The actual game screen once a setup goes active (future Svelte SPA work).
- Real email delivery for invites (link-only for now).
- Timeout/expiry/reminder handling for slow-to-respond invitees.
- Host-controlled or player-chosen nation assignment (random only) — a
  reasonable later feature, not a redesign to add.
- Minimum player count validation beyond "the host is always in."

## Status

Nothing implemented yet. This document reflects direction agreed on before
any of it exists, to be picked up in pieces rather than all at once.
