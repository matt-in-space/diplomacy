# Roadmap

## Status

Movement and retreats are fully implemented end to end: order submission and
validation, adjudication, and phase transitions. Supply-center ownership
tracking and the adjustment query surface (who owes what, where they may
build) are in place; build/disband order types and their resolution are not
started. There is no web/HTTP layer — the
application layer (`application/gameplay`) exists with in-memory repositories
only, driven directly by its own tests and `cmd/server`'s minimal wiring.

## Engine (`core/`)

### Movement — done

- Order submission and validation (Hold, Move, Support Hold, Support Move,
  Convoy) — `core/game`.
- Adjudication via a recursive backtracking resolver (Kruijswijk's "The Math
  of Adjudication"): holds, moves, bounces/standoffs, support, support-cut,
  convoys, convoy disruption by dislodgement, and circular movement —
  `core/adjudicator`.
- See `docs/adjudication-resolver.md` for how the resolver actually works.
- **Not done:** convoy paradoxes (DATC 6.F, e.g. Pandin's paradox). The
  Szykman backup rule is the intended fix but isn't wired to the right
  dependency-cycle node yet — see `docs/adjudication-enhancements.md`. One
  test (`TestResolve_ConvoyParadox`) documents the intended outcome and is
  skipped until this lands.

### Retreats — done

- `Unit.DislodgedFrom` marks a unit that needs a retreat order once movement
  resolves.
- `LegalRetreats` derives legal destinations from the prior movement
  `Resolution`: excludes occupied provinces, standoff provinces, and the
  attacker's origin province — unless the attacker arrived by convoy.
- Retreat order submission (`RetreatOrder`, `DisbandOrder`) through the same
  phase-dispatching `SubmitOrder` entry point movement orders use.
- Retreat resolution: a unit with no submitted order defaults to disband
  (mirroring movement's default-to-hold); two or more units retreating to the
  same province are all disbanded — a retreat conflict has no "stay put"
  fallback the way a movement bounce does.

### Adjustments — done except civil disorder

- `Game.SupplyCenterOwners` tracks ownership, seeded at game creation from the
  map's home centers and updated by the `UpdateOwnership` phase, which runs
  once after Fall retreats resolve (not Spring) and never runs otherwise —
  the season-gating is structural, encoded in `Turn.Next()`, not a runtime
  check.
- `Game.AdjustmentBalance` and `Game.LegalBuildProvinces` derive what a
  nation owes (builds/disbands) and where it may build, the same way
  `LegalRetreats` derives retreat legality — no separate stored state, since
  both are plain reads of `SupplyCenterOwners`/`Units` that can't drift from
  their sources.
- `Order` now requires only `Nation()`; `UnitOrder` (`Order` plus `Unit()`)
  names the subset — every existing order type — that acts on a unit already
  on the board. `Game.Orders` is a slice rather than a `map[UnitID]Order`, so
  a unit-less order has somewhere to live; `Game.OrderFor`/`Game.UnitOrders`
  derive the by-unit lookups movement, retreats, and the adjudicator need.
  This clears the way for a `BuildOrder`, which creates a unit rather than
  naming one.
- `BuildOrder` (`core/game/build_order.go`) creates a unit at a home center
  the nation owns and controls; it satisfies `Order` but not `UnitOrder`,
  since it names no existing unit. `SubmitOrder` accepts both `BuildOrder`
  and `DisbandOrder` during `AcceptAdjustments`, checking each against the
  *sign* of `AdjustmentBalance` (build needs positive, disband needs
  negative) — the same kind of current-state check `submitRetreatOrder`
  already does with `unit.Dislodged()`. `DisbandOrder` is reused as-is from
  retreats; only the precondition differs by phase.
- `BeginAdjustmentResolution`/`nationsAwaitingAdjustments` gate the phase
  (mirroring retreats); `adjudicator.ResolveAdjustments` mints `UnitID`s for
  builds and computes disbands; `CompleteAdjustmentResolution` applies both.
  A nation may submit fewer builds than its balance allows (waived, legal);
  submitting more than its balance, in either direction, is an error.
- **Not done:** civil disorder. A nation with a negative balance that
  doesn't submit enough disbands to cover it should have units
  auto-selected for removal (conventionally farthest from home, with a
  tiebreak) — instead `ResolveAdjustments` currently returns a hard error
  naming the nation and the shortfall. This also means a passive/departed
  player can stall a game's processing rather than the game working around
  them.
- **Not done at all:** victory condition (18 supply centers) and
  game-over/draw handling — nothing currently detects that a game has ended.
- **Not done:** elimination. A nation reduced to zero units and zero supply
  centers isn't removed from `Assignments` or otherwise marked inactive;
  every phase's "await commitment" gate still expects it to commit like any
  other nation.

## Application layer (`application/gameplay`)

- `GameplayService` drives phase transitions (`ProcessGame`/
  `processGameStep`) and exposes `SubmitOrder`/`CommitOrders` commands.
- `GameRepository`, `GameMapRepository`, and `PlayerRepository` are
  interfaces defined here (where they're consumed), per the rule already
  written in `docs/architecture.md`. Their only implementations so far are
  in-memory, and now live in `infrastructure/memory` rather than alongside
  the interfaces — no database-backed implementation yet.
- No history/audit trail. `Game` intentionally does not persist orders and
  outcomes past the phase that produced them — `LastOrderResolution` and
  `LastRetreatResolution` exist because retreat-legality *rules* need them
  functionally, not for display. A persisted history store belongs at the
  application layer once there's a UI to serve it (see `docs/architecture.md`).
- A `web` package now exists and `cmd/server` actually serves HTTP: a
  stdlib `net/http.ServeMux`, one route (`GET /`, a hello-world page
  rendered via `html/template`), and Pico.css vendored and served from an
  embedded static filesystem (`/static/`) — matching the styling and
  "embed everything into one binary" decisions in `docs/user-experience.md`.
  Flat for now, no `web/handlers`/`web/middleware` split yet — that's
  structure worth adding once auth gives it enough route surface to justify
  it. `GameplayService` is still constructed in `main.go` but not wired to
  any handler. Fixing this also resolved `cmd/server`'s longstanding
  relative-path panic: the map fixture is now loaded via `//go:embed` in
  `core/gamemap` (`gamemap.WesternEurope()`) instead of `os.ReadFile`, so
  `go run ./cmd/server` works from the repo root. Direction for the rest of
  the frontend and API (Svelte SPA, SVG map rendering, REST plus a
  real-time channel for live updates) is captured in
  `docs/user-experience.md`; none of that is implemented yet. The build
  order for closing that gap is below.

## Next up: auth → game SPA → database

Engine work is paused (see the "Not done" items above) in favor of building
the application around it. The order is deliberate — each phase is chosen
to avoid throwaway work in the next one, not just picked for convenience:

1. **Authentication, backed by in-memory repositories.** Self-hosted
   email + password (bcrypt-hashed), chosen over OAuth for now: no external
   app registration, fully self-contained, most learning value.
   - **Landed:** the credential-bearing `Player` type and its repository
     moved to a new `application/auth` package rather than extending
     `core/game.Player` — the domain layer only ever needs a `PlayerID` to
     assign nations to, never an email address or password hash.
     `auth.Player` has `ID`, `Email`, `DisplayName` (shown to other
     players; `Email` is login-only, never exposed to them),
     `PasswordHash`, `EmailVerified`, `CreatedAt`, `PasswordChangedAt`.
     `auth.PlayerRepository` adds `GetPlayerByEmail` alongside
     `CreatePlayer`/`GetPlayer`/`SavePlayer`, backed by
     `infrastructure/memory.PlayerRepository` (email uniqueness enforced
     via a secondary index, `PasswordHash` defensively cloned at repository
     boundaries the same way `Game` already is). `core/game.Player` (the
     struct) was removed as dead code in the same pass — nothing in
     `core/game` ever used more than `PlayerID`. `GameplayService` no
     longer takes a `PlayerRepository` at all; it never actually used it.
     `auth.Session` (`Token`, `PlayerID`, `CreatedAt`, `ExpiresAt`) and
     `auth.SessionRepository` also landed — chosen over a stateless signed
     cookie specifically because a stateless design can't support real
     revocation (logout, "log out everywhere," invalidating sessions on
     password change all need server-side state regardless). `GetSession`
     treats an expired token the same as an unknown one; `DeleteSession`
     and `DeleteSessionsForPlayer` are idempotent — deleting an
     already-gone session is a normal case, not a bug. Backed by
     `infrastructure/memory.SessionRepository`; unlike `Player`, `Session`
     has no reference-type fields, so no defensive cloning is needed at the
     repository boundary. Token generation (`crypto/rand`, never anything
     sequential) is the future login handler's job, not the repository's.
   - **Landed:** `auth.Service` (`application/auth/service.go`) — signup
     (bcrypt hashing, minimal validation: email shape not deliverability,
     display name non-empty, 8-char password minimum with no forced
     complexity, matching current NIST guidance), login (constant-time
     against an unknown email — still runs a bcrypt comparison against a
     fixed dummy hash before returning, so response timing doesn't leak
     which emails are registered; wrong password and unknown email both
     return the same `ErrInvalidCredentials`), logout, and `Authenticate`
     (session token → `Player`, used by request middleware). This is the
     project's first external dependency (`golang.org/x/crypto/bcrypt`) —
     nothing before this needed anything outside the standard library.
     `web/` gained `POST /signup`, `POST /login`, `POST /logout`
     (logout is POST-only deliberately — a state-changing action shouldn't
     be reachable by a plain link or GET prefetch), a global
     "populate current player from the session cookie, but never block the
     request" middleware (`web/session_middleware.go` — optional auth, not
     a login gate; nothing yet requires being logged in to reach it), and
     an independent flash-message mechanism (`web/flash.go`) — deliberately
     not tied to `auth.Session`, since flash needs to work for anonymous
     flows too ("account created, please log in" happens before any session
     exists). `cmd/server/main.go`'s repositories stopped being
     constructed-and-discarded and are wired for real for the first time.
   - **Landed:** `GET /signup`/`GET /login` render real forms
     (`web/templates/signup.html`/`login.html`), and a shared
     `web/templates/layout.html` now owns the `<head>`, a Pico.css classless
     `<nav>` (bare `<nav>`/`<ul>` — no extra classes needed), and flash
     rendering — the point flagged as "worth deciding once three templates
     want the same boilerplate" arrived and got built. The nav shows the
     player's `DisplayName` and a logout button (a form, not a link — logout
     stays POST-only) when logged in, login/signup links otherwise. Each
     page is its own isolated `*template.Template` (`web/templates.go`'s
     `parsePage`, combining `layout.html` with exactly one page) rather than
     one shared parse of every `.html` file — parsing them all together
     would have every page's `{{define "content"}}` collide in one
     namespace, so only the last-parsed page's content would ever render.
     `redirectIfAuthenticated` (`web/session_middleware.go`) sends an
     already-logged-in visitor away from `/signup`/`/login` back to `/`,
     applied to the `GET` (view the form) routes only. `web/auth.go`'s POST
     handlers renamed `handleSignup`/`handleLogin` →
     `handleSignupSubmit`/`handleLoginSubmit` now that GET counterparts
     exist. CSRF is already covered for all three POST routes by
     `SameSite=Lax` on the session/flash cookies (set in Part 1) — nothing
     new needed for real browser-submitted forms. No per-field inline
     validation or refilled form values on a failed submit, and no
     success/error color distinction on flash messages yet (the
     `class="{{.Kind}}"` hook exists in the markup; the CSS doesn't) —
     both deliberate, matching "very simple looking."
2. **The game SPA**, still against in-memory repositories, built on top of
   a real session from step 1 rather than a stand-in identity — so nothing
   here needs revisiting once real auth exists, because it already will.
   Svelte scoped to the in-game screen, SVG map rendering, REST plus the
   real-time channel — all per `docs/user-experience.md`.
3. **Database and infrastructure** — Postgres via Podman, `pgx`/`pgxpool`,
   `sqlc`, migrations. Deliberately last: `GameRepository`,
   `PlayerRepository`, `GameMapRepository`, and the new `SessionRepository`
   are all interfaces specifically so this can be deferred until the shapes
   of what's actually being persisted are proven out by steps 1-2, instead
   of designing a schema speculatively. Swapping in a Postgres-backed
   implementation touches only new files, not anything built in steps 1-2.

## Known issues

None currently open.

## Open design questions

- **Orders subpackage.** Grouping the order files (`hold_order.go`,
  `move_order.go`, etc.) into a `core/game/orders` subpackage was explored and
  hits a real wall: twelve `*Game` methods live in those files (including both
  exported entry points), and there's a genuine import cycle — `Game.Orders`
  needs the `Order` type, while every validator needs `Game`/`Unit`. A plain
  rename (`order_hold.go`, `order_move.go`, ...) solves the "hard to scan"
  problem without fighting Go's package model; a real subpackage split would
  mean hoisting `Unit`/`UnitID` into a third leaf package and converting every
  validator into a free function taking `*game.Game`. Undecided.
## Resolved design questions

- **Repository location.** In-memory repository implementations
  (`MemoryGameRepository`, `MemoryPlayerRepository`, `MemoryGameMapRepository`)
  moved out of `application/gameplay` into `infrastructure/memory`
  (`GameRepository`, `PlayerRepository`, `GameMapRepository`, "Memory"
  prefix dropped now that the package name carries that meaning), matching
  `docs/architecture.md`'s proposed layout and its stated rule that
  interfaces belong with the application code that consumes them, while
  implementations belong in infrastructure. The interfaces, `StoredGame`,
  and the sentinel errors (`ErrGameNotFound`, `ErrPlayerNotFound`, etc.)
  stayed in `application/gameplay` — they're the contract, not the backend.
  One wrinkle: `application/gameplay`'s own *internal* tests can't import
  `infrastructure/memory` (it imports `application/gameplay` to implement
  its interfaces, so an internal test doing the reverse is a real import
  cycle, not just a style question — confirmed by `go vet`). The one test
  that needed real Get/Save round-tripping fidelity (rather than the
  call-counting fakes used elsewhere) moved to the external
  `gameplay_test` package, which has no such restriction.

## Documentation debt

Several docs under `docs/` were written before or during early implementation
and describe designs that were later changed. Flagging so they aren't mistaken
for current behavior — none of this was reconciled as part of writing this
roadmap:

- `docs/map-notation.md` and `docs/adjudication-pipeline.md` reference
  `Game.Positions` and `Game.FleetCoasts`, which no longer exist.
  `Unit.ProvinceID` and `Unit.Coast` are now the single source of truth.
- `docs/order-resolution.md` and `docs/orders.md` describe an earlier
  `Resolution`/`Order` shape (separate `OrderResult`/`MoveResult`/
  `DislodgedUnit` slices) that isn't what was built. The actual shape is
  `Resolution map[UnitID]Outcome` — `docs/adjudication-resolver.md` is much
  closer to what's actually there.
- `docs/architecture.md` describes an aspirational production architecture
  (durable jobs, transactional outbox, events, HTTP layer, Postgres) — none of
  it exists yet. Read it as a plan, not a description of the current system.
- `docs/testing.md` proposes a shared `maptest` builder package; the codebase
  instead uses JSON fixtures (`core/gamemap/testdata/`) plus local
  `_test.go` helpers per package. Not reconciled.
