# Game Setup Flow

## Purpose

This document captures the design for getting from "I want to play a game
with some friends" to an actual `core/game.Game` that's ready for orders —
listing your games, creating one, sharing a join link, and kicking off. Same
spirit as `docs/user-experience.md`: write decisions down as they're made,
flag what's still open, let it evolve as it's actually built.

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

## Decisions

- **Joining: one shared code per setup, not a per-person invite.** Early
  design had a per-email `Invite` record with its own
  pending/accepted/declined state machine. In practice the email was never
  load-bearing — explicitly "a label, not access control," since anyone
  holding the link could accept regardless of which account they used. That
  meant the state machine was wrapped around a fact nobody actually checked.
  Collapsing to one `GameSetup.InviteCode`, shared with everyone the host
  wants to invite, keeps the same "whoever holds the link can join" model
  (still no email-matching failure mode — a host typo or an invitee using a
  different account email no longer strands anything) while deleting the
  per-person bookkeeping entirely. `docs/user-experience.md` already
  anticipated the link-based shape: *"an invite link needs to open cold...
  without hydrating a JS bundle first."*
- **Membership is a plain list on the setup.** `GameSetup.PlayerIDs` holds
  everyone in the lobby, host included — no separate repository. A join
  still needs to be an atomic write rather than a whole-object save (two
  people using the link within milliseconds of each other must not be able
  to lose each other's slot, or both pass a capacity check that only had
  room for one), so `GameSetupRepository` has one narrow method,
  `AddPlayerToGameSetup`, alongside the generic `SaveGameSetup` — not a
  break in convention, `GameRepository.SaveGame(g, expectedVersion)` is
  already a non-uniform write method for the same class of reason.
- **Joining is capped at the map's nation count.** A shareable link is
  forwardable in a way per-person invites weren't — one paste into a group
  chat can bring more clicks than there are nations. Rejecting the joiner
  past capacity is better than letting them into the lobby and having them
  silently get no nation at kickoff.
- **Starting: the host alone decides when.** There is no minimum
  player-count gate — a host judgment call, not a validated rule, for v1.
  Capacity is already enforced at join time, so `StartGame` can never
  receive more players than nations to assign.
- **Nation assignment: random shuffle at kickoff.** Everyone in `PlayerIDs`
  (the host included) is randomly assigned to the map's nations; fewer
  joiners than nations just leaves the remainder vacant (already
  supported). Host-controlled or player-picked assignment is a reasonable
  later feature, not a redesign to retrofit.
- **Kickoff "notifications" = visibility, not delivery.** For v1, starting
  a game just needs to make it show up on the players' `/games` list — no
  real email or push. Real-time "your game just started" notification is a
  natural fit for the WebSocket channel already planned for the in-game
  SPA, not something to build now.
- **The host is seeded into `PlayerIDs` at creation**, not via the join
  flow — they don't use their own invite link (doing so is a harmless
  no-op, since joining is idempotent).
- **`GameSetup.Status` is computed, not stored.** Only `CancelledAt
  *time.Time` is genuinely non-derivable state. Whether a setup is Active is
  answered by checking whether a `core/game.Game` already exists for its ID
  — the same "don't duplicate derivable state" reasoning already applied to
  `AdjustmentBalance`/`LegalRetreats` elsewhere in this codebase, and it
  avoids a real two-write inconsistency risk (`StartGame` creating the
  `Game` but failing before a separately-stored status flag got flipped).
- **`GameSetup.HostID` doubles as the owner handle for administering a
  running game later** — not a new concept, just the existing field's
  authority extended past kickoff. Scoped for now to exactly one privilege
  once a setup goes active: force-ending a stuck game (flipping
  `Turn.Phase` to `Completed`, which the domain already defines but nothing
  currently has a path to reach — no victory condition, no draw mechanism,
  no civil disorder yet). That action (`EndGame` or similar) is out of
  scope for this document; it belongs with administering an active game,
  not with setup.
- **Cancelling a pending setup** is in scope alongside starting (host-only,
  small) — cheap enough that a stalled lobby isn't permanently stuck with
  no way out.

## Design

### `application/lobby` (implemented)

```go
type Status string
const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusCancelled Status = "cancelled"
)

type GameSetup struct {
	ID          game.GameID   // becomes the real Game's ID once started — no remapping
	MapID       gamemap.MapID
	HostID      game.PlayerID   // admin authority only — start, cancel; membership is PlayerIDs
	InviteCode  string          // the credential — one per setup, shared with everyone who gets the link
	PlayerIDs   []game.PlayerID // everyone in the lobby; the host is seeded in at index 0
	CreatedAt   time.Time
	CancelledAt *time.Time // nil = not cancelled; the one fact that can't be derived
}
```

One repository (interface in `application/lobby`, in-memory implementation
in `infrastructure/memory`, same split as every other resource):

```go
type GameSetupRepository interface {
	CreateGameSetup(ctx context.Context, setup *GameSetup) error
	GetGameSetup(ctx context.Context, id game.GameID) (*GameSetup, error)
	GetGameSetupByInviteCode(ctx context.Context, code string) (*GameSetup, error)
	SaveGameSetup(ctx context.Context, setup *GameSetup) error
	// AddPlayerToGameSetup is the one atomic write: append-if-absent,
	// capacity-checked, so a join can't be lost or overrun a slot.
	AddPlayerToGameSetup(ctx context.Context, gameID game.GameID, playerID game.PlayerID, capacity int) error
	// hosted by or joined into — the host is always in PlayerIDs, so one
	// membership scan answers both.
	ListGameSetupsForPlayer(ctx context.Context, playerID game.PlayerID) ([]*GameSetup, error)
}
```

`Service`:

```go
type Service struct {
	setups   GameSetupRepository
	games    gameplay.GameRepository
	maps     gameplay.GameMapRepository
	gameplay *gameplay.GameplayService
}

func (s *Service) StatusFor(ctx context.Context, setup *GameSetup) (Status, error)
func (s *Service) CreateGameSetup(ctx context.Context, hostID game.PlayerID, mapID gamemap.MapID) (*GameSetup, error)
func (s *Service) JoinGameSetup(ctx context.Context, code string, playerID game.PlayerID) (*GameSetup, error) // idempotent, capacity-checked
func (s *Service) StartGame(ctx context.Context, gameID game.GameID, requesterID game.PlayerID) error // host-only; random nation shuffle; calls gameplay.CreateGame
func (s *Service) CancelGameSetup(ctx context.Context, gameID game.GameID, requesterID game.PlayerID) error
```

`auth.PlayerRepository` is deliberately *not* a `lobby.Service` dependency —
nothing in the join/start/cancel logic needs to look up an account; showing
a display name in the lobby view is a rendering concern for the web layer.

`StartGame`'s core logic:

```go
nations := slices.Clone(gm.Nations)
players := slices.Clone(setup.PlayerIDs)
rand.Shuffle(len(nations), ...)
rand.Shuffle(len(players), ...)
assignments := map[gamemap.NationID]game.PlayerID{}
for i, playerID := range players {
	if i >= len(nations) { break } // unreachable in practice — capacity is enforced at join time
	assignments[nations[i]] = playerID
}
_, err := s.gameplay.CreateGame(ctx, setup.ID, setup.MapID, assignments)
```

### `application/gameplay` — new use cases

```go
// create_game.go (implemented)
func (s *GameplayService) CreateGame(ctx context.Context, id game.GameID, mapID gamemap.MapID, assignments map[gamemap.NationID]game.PlayerID) (StoredGame, error)
// list_games.go (not yet built — needed for the /games page below)
func (s *GameplayService) ListGamesForPlayer(ctx context.Context, playerID game.PlayerID) ([]StoredGame, error)
```

`lobby.Service` calls *into* `GameplayService.CreateGame` at kickoff rather
than constructing games itself — `lobby` decides who/when, `gameplay` still
owns how a game actually gets created and persisted. `GameSetup.ID` is
reused verbatim as the resulting `Game.ID`, so there's no remapping step and
`StatusFor`'s "does a `Game` exist for this ID" check works without any
extra stored linkage.

`GameRepository` will need `ListGamesForPlayer` for the `/games` list below
— its first new method since it was built. In-memory: a plain linear scan
over stored games' `Assignments`. Postgres will eventually want a real
indexed query against the `games` metadata table instead of scanning
JSONB — already the documented plan, not a new decision.

### `web` — new routes (not yet built)

```
GET  /games                    require login — lists: active games (GameplayService.ListGamesForPlayer),
                                setups you host or joined (GameSetupRepository.ListGameSetupsForPlayer)
GET  /games/new                require login — create-game form (map choice trivial: one option today)
POST /games                    creates a GameSetup, redirects to its lobby page
GET  /games/{id}               require login — pending: lobby view (invite link, joined-players list, start/cancel, host-only actions);
                                active: minimal stub ("game started" — the real game screen is future work)
POST /games/{id}/start         host-only
POST /games/{id}/cancel        host-only
GET  /join/{code}              require login (redirect through /login?next=... if not) — joins the caller, then redirects to /games/{id}
```

The old per-invite `GET/POST /invites/{code}/...` routes are gone along with
the `Invite` entity — joining is a single `GET /join/{code}` that both
resolves the code and performs the join in one step, since there's nothing
left to accept or decline.

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
never `//` or a full URL), to avoid becoming an open redirect. Worth
deciding when this gets built: `handleSignupSubmit` doesn't currently carry
`next` through at all, so someone who clicks a join link with no account
yet would land on `/` after signing up rather than back at `/join/{code}`.

New templates, through the existing `parsePage("templates/<name>.html")` +
shared-layout pattern: `games.html` (the list), `games_new.html`,
`game_setup.html` (lobby detail — shows the invite link for the host to
copy).

## Out of scope

- The actual game screen once a setup goes active (future Svelte SPA work).
- Real email delivery — the invite code is shared as a link, out of band,
  by whoever's hosting.
- Invite code rotation.
- Host-controlled or player-chosen nation assignment (random only) — a
  reasonable later feature, not a redesign to add.
- Minimum player count validation beyond "the host is always in."
- The active-game `EndGame` owner action (force-completing a stuck game) —
  administering a running game, not game setup.

## Status

Backend plumbing implemented: `application/lobby` (types, repository
interface, service) and its `infrastructure/memory` implementation, plus
`application/gameplay.CreateGame`. The `web` layer — routes, templates,
`cmd/server/main.go` wiring — is not yet built.
