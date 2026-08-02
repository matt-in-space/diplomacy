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
  convoys, convoy disruption by dislodgement, circular movement, and convoy
  paradoxes (Szykman rule) — `core/adjudicator`.
- See `docs/adjudication-resolver.md` for how the resolver actually works.

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

### Adjustments — in progress

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
- Still needed: `BuildOrder` itself and its validation, `AcceptAdjustments`/
  `ResolveAdjustments` phase wiring, and forced disbands when a nation
  under-orders. Adjustment orders are validated as a set against a nation's
  `AdjustmentBalance`, unlike movement/retreat orders which are each
  independently valid — `SubmitOrder`'s one-at-a-time shape may not fit as-is.
- Victory condition (18 supply centers) and game-over/draw handling.

## Application layer (`application/gameplay`)

- `GameplayService` drives phase transitions (`ProcessGame`/
  `processGameStep`) and exposes `SubmitOrder`/`CommitOrders` commands.
- `GameRepository`, `GameMapRepository`, and `PlayerRepository` are in-memory
  only — no database-backed implementation yet.
- No history/audit trail. `Game` intentionally does not persist orders and
  outcomes past the phase that produced them — `LastOrderResolution` and
  `LastRetreatResolution` exist because retreat-legality *rules* need them
  functionally, not for display. A persisted history store belongs at the
  application layer once there's a UI to serve it (see `docs/architecture.md`).
- No web/HTTP layer. `cmd/server/main.go` only constructs the service and
  loads the test map.

## Known issues

- `cmd/server/main.go` loads its map fixture with a path relative to the
  working directory, not the source file, so `go run ./cmd/server` from the
  repo root panics. Run it from inside `cmd/server/` for now
  (`cd cmd/server && go run .`).

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
- **Repository location.** `application/gameplay` currently holds both use-case
  logic and the repository interfaces/in-memory implementations in one
  package. Moving repositories into a dedicated location (matching
  `docs/architecture.md`'s proposed infrastructure layout) is still open.

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
