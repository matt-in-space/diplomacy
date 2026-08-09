# Game Screen UI

## Purpose

This document tracks ideas and decisions for the actual game screen — the
map, order building, turn status, everything that happens once a
`GameSetup` has started and `web/games.go`'s `handleGame` hands off to the
frontend. Same spirit as `docs/game-setup.md`: write decisions down as
they're made, flag what's still open, let it evolve as it's actually
built. Nothing here is binding — it's a place to track ideas, not a spec
to hold ourselves to.

This supersedes `docs/user-experience.md`'s "In-game frontend (Svelte SPA)"
section and `docs/game-setup.md`'s "future Svelte SPA" references for the
game screen specifically — Svelte isn't the plan. Both docs are due a
correction pass; not done here to keep this doc scoped to the game screen
itself.

## Frontend approach

Vanilla JS (`web/static/js/*.mjs`), no build step, per the existing
`main.mjs`/`mapData.mjs`/`mapRender.mjs`. Decision: stay vanilla until
cross-cutting state (one change needing to update several panels at once —
e.g. selecting a unit highlighting it on the map, in a list, and in an
order panel simultaneously) makes hand-wiring DOM updates genuinely
unmanageable. If that happens, the fallback is a hand-rolled reactive
store — the project briefly had exactly this (`createStore<T>`,
subscribe/notify) before deleting it as unused — not reaching for a
framework.

## Persistent turn/phase status

An always-visible bar showing e.g. "Spring 1902, Accepting Orders."
Backed by real data already: `core/game.Turn{Season,Phase,Year}` — same
formatting precedent as `web/home.go`'s `formatTurn`, used for the home
page's games list.

## Nation sidebar

Each nation in the game listed, each assigned a distinct color. Data:
`gamemap.GameMap.Nations []NationID`.

Hovering a nation's name shows stats and an entry point into the
messaging area. Messaging itself isn't built yet — this is a placeholder/
disabled affordance until it exists, not something to fake.

Open question: are colors fixed per nation (classic Diplomacy convention —
Austria red, England navy, France light blue, Germany black/grey, Italy
green, Russia white, Turkey yellow) or assigned per-game? Our current
subset map only has `eng`/`fra`, so this isn't urgent, but worth deciding
before the full board is in play.

## Province ownership tinting

Idea: tint each province the color of the nation that currently "owns"
it, tracked as the last unit to move through it. Explicitly **not** a real
Diplomacy rule — only supply centers are actually owned
(`core/game.Game.SupplyCenterOwners`, `core/game/game.go:28`, updated by
`CompleteOwnershipUpdate` in `core/game/ownership.go`). This would be a UI
convenience/house rule layered on top, and the backend doesn't track
general province occupancy at all today.

Open questions to resolve before any backend work:

- Does ownership persist once set, or reset each turn?
- Does a bounced/failed move count as "moving through"?
- On dislodge/retreat, does the dislodged nation keep the province it was
  forced out of, or does the dislodging nation take it immediately?

Rough shape, not binding: something like
`Game.ProvinceOwners map[gamemap.ProvinceID]gamemap.NationID`, updated
wherever a unit successfully occupies a province — parallel to how
`SupplyCenterOwners` already works, just not restricted to supply centers.

## Unit list

A sidebar (opposite side, or a toggle) listing your own units — type and
location. Data: `core/game.Unit{ID,NationID,ProvinceID,Type,Coast,
DislodgedFrom}` already has everything needed. Open question: does this
list show only your own units, or everyone's (already visible on the map
regardless)?

## Selection & order building

Click a unit — on the map or in the list — to see its details and add or
edit an order. No backend blocker here; `application/gameplay` order
submission already exists. This is the piece most worth designing
carefully before building, not an engineering problem:

- **Support** is the known-hard case. Supporting your own unit is simple
  (pick the unit). Supporting another nation's unit requires declaring
  what you *expect* it to do — hold, or move to a specific province —
  because a support order only succeeds if it matches what that unit
  actually does. You're often guessing at a rival's intent.
- **Convoy** likely has a similar expectation-declaring shape and should
  reuse whatever pattern the support flow lands on, rather than being
  designed independently.

## Submitting orders / waiting state

A submit button. Once submitted (and others haven't finished), a waiting
state: "waiting on other players, turn ends in X."

Real gap: there is no deadline/timer concept anywhere in the backend
today. This isn't only a UI countdown — something has to actually resolve
a turn on a clock even if not everyone has submitted, which is a real
scheduling/async-processing feature, not a widget.

Rough shape, not binding: a deadline timestamp on `Game` or `GameSetup`,
plus some mechanism (background ticker, or a check triggered per-request)
that forces `ProcessGame` once it passes.

## Post-resolution results playback

After `AcceptOrders` resolves, each player gets their own pass through
what happened — stepping through moves one at a time, not just landing on
the new state.

Real gap: `core/game.Unit` only tracks current position
(`core/game/unit.go`) — nothing records where a unit was before a
resolution. Rough shape, not binding: could come from diffing before/after
`Game` snapshots if those end up stored anywhere, or from resolution
explicitly recording a per-turn move log. Not decided which.

## Sequencing

1. **Look OK first** (this document's original prompt): build the pieces
   that already have real data — turn/phase bar, nation sidebar, unit
   list — against that real data. Build the three "iceberg" pieces
   (ownership tinting, deadline countdown, results playback) against
   placeholder/mock data for now, without committing backend work yet.
2. **Functionality**: order submission flow first (no backend blocker,
   just design + build). Then the iceberg items, each likely warranting
   its own scoped discussion/plan when it's picked up, same as everything
   else in this project.

## Open questions

- Nation color assignment: fixed per nation, or per-game?
- Province ownership semantics (persistence, bounces, retreats) — see
  above.
- Deadline mechanism — see above.
- Results-playback data source — see above.
- Unit list scope: own units only, or everyone's?
