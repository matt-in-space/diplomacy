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
- **Starting requires the lobby to be full.** Superseding the earlier "host
  judgment call, no minimum" decision: `StartGame` now rejects
  (`ErrGameSetupNotFull`) unless every nation on the map has a joined
  player. The web layer reflects this before the host even tries — the
  Start button is `disabled` and shows live progress ("Start Game (1/2)")
  until the lobby fills — but the enforcement lives in the service, not
  just the UI, since a disabled HTML attribute is trivially bypassable by
  posting to the route directly. Capacity is already enforced at join time,
  so `StartGame` can never receive more players than nations to assign —
  this new check only ever needs `>=`, not exact-match, math.
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

### `web` — routes

```
GET  /games/new                 require login — create-game form (implemented; one map option today)
POST /games                     creates a GameSetup, redirects to /games/{id}/lobby (implemented)
GET  /games/join                require login — paste-a-code form (implemented)
POST /games/join                joins via lobby.Service.JoinGameSetup, redirects to
                                 /games/{id}/lobby on success, or back to /games/join with
                                 a specific flash message on failure (implemented)
GET  /games/{id}/lobby          require login — status, host, player count/table, and (while
                                 pending) a Start button disabled until the lobby is full
                                 (implemented)
POST /games/{id}/start          host-only, rejects unless full (ErrGameSetupNotFull) or the
                                 setup isn't open (ErrGameSetupNotOpen); redirects to
                                 /games/{id} on success, back to /games/{id}/lobby with a
                                 flash on failure (implemented)
GET  /games/{id}                require login — Active: a blank placeholder page (the real
                                 game screen is future work); Pending/Cancelled: redirects
                                 to /games/{id}/lobby, so this is a stable "go to this game"
                                 link regardless of state (implemented)
GET  /games                     not yet built — needs GameplayService.ListGamesForPlayer
POST /games/{id}/cancel         not yet built
GET  /join/{code}               deliberately deferred — the code is surfaced today as plain
                                 text in the lobby's "Share this code" callout, not a link,
                                 so a clickable /join/{code} URL has no real caller yet
```

`/games/{id}` is viewable by any logged-in user who knows the ID — same
permissiveness `/games/{id}/lobby` already had, no check that the viewer is
actually a participant. Noted as a known, pre-existing gap rather than
silently ignored; tightening it is separate work if/when it matters.

`requireAuthentication` (`web/session_middleware.go`) is built — the
blocking counterpart to `withCurrentPlayer`'s non-blocking style, redirects
to `/login?next=<path>` if there's no current player. `handleLoginForm`
renders `next` into a hidden field; `handleLoginSubmit` redirects there on
success, validated through `safeRedirectTarget` (same-site relative path
only — starts with exactly one `/`, never `//` or a full URL) so it can't
become an open redirect. `handleSignupSubmit` also carries `next` through
now (appended onto its redirect to `/login`), and the login/signup pages'
cross-links to each other preserve it too — closing the gap this document
used to flag ("doesn't currently carry `next` through at all").

Templates, through the existing `parsePage("templates/<name>.html")` +
shared-layout pattern: `games_new.html`, `games_join.html`,
`game_setup_lobby.html`, `game.html` (the blank active-game placeholder). A
`games` list page is not yet built, per the routes above.

`lobby.Service.JoinGameSetup` normalizes the submitted code (trim,
uppercase) before looking it up — the code alphabet
(`internal/random`'s `codeAlphabet`) is uppercase-only, and this is a value
a human actually types, unlike every other ID in the system. `POST
/games/join` maps `JoinGameSetup`'s sentinel errors
(`ErrGameSetupNotFound`/`ErrGameSetupFull`/`ErrGameSetupNotOpen`) to
specific flash copy rather than echoing the raw wrapped error back. `POST
/games/{id}/start` does the same for `StartGame`'s errors
(`ErrNotHost`/`ErrGameSetupNotFull`/`ErrGameSetupNotOpen`).

## Out of scope

- The actual game screen once a setup goes active (future Svelte SPA work)
  — `/games/{id}` is a blank placeholder until then.
- Real email delivery — the invite code is shared as a link, out of band,
  by whoever's hosting.
- Invite code rotation.
- Host-controlled or player-chosen nation assignment (random only) — a
  reasonable later feature, not a redesign to add.
- The active-game `EndGame` owner action (force-completing a stuck game) —
  administering a running game, not game setup.
- Per-game participant access control on `/games/{id}`/`/games/{id}/lobby`.

## Status

Backend plumbing implemented: `application/lobby` (types, repository
interface, service, including `GetGameSetup` and `ReadyToStart` read-only
passthroughs for the web layer) and its `infrastructure/memory`
implementation, plus `application/gameplay.CreateGame`.
`cmd/server/main.go` wires it all up — `lobby.Service` is no longer
discarded.

Web: create (home → `/games/new` → `POST /games` → `/games/{id}/lobby`),
join (home → `/games/join` → `POST /games/join` → `/games/{id}/lobby`),
and start (lobby's Start button, disabled until full → `POST
/games/{id}/start` → `/games/{id}`) are all implemented end to end, gated
by `requireAuthentication` with a working `next` round-trip through both
login and signup. The lobby page resolves display names for every player
via `auth.Service.GetPlayer`, rather than showing raw `PlayerID`s. Still
open: the `/games` list, Cancel on the lobby page, and `/join/{code}` as a
clickable link.
