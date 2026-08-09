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

Stub built: `.turn-status` in `web/templates/game.html`, a brass-bordered
plaque overlaying the map's top-left corner (`web/static/game.css`).
Hardcoded to "Spring 1902, Accepting Orders" — not yet wired to real data.

Not yet done: backed by real data. `core/game.Turn{Season,Phase,Year}` has
everything needed — same formatting precedent as `web/home.go`'s
`formatTurn`, used for the home page's games list.

## Map pan/zoom

Built. `web/static/js/mapViewport.mjs` — mouse-drag panning, scroll-wheel
zoom-to-cursor, and arrow-key panning, all clamped so the view can never
show area outside the map's real content bounds or zoom out further than
"the whole map fits" (`clampViewBox`/`zoomViewBox`/`panViewBox`, unit
tested in `_tests/mapViewport.test.mjs`). `0` resets to the fit view.

Decisions: arrow keys, not WASD (the conventional "pan a view" idiom,
versus WASD's real-time-action-game connotation, which doesn't fit a
turn-based board game). Gesture-only for this pass, no on-screen +/−/reset
buttons — easy to add later if discoverability turns out to matter. Uses
Pointer Events (not legacy mouse events) so single-finger touch dragging
works for free later; pinch-zoom and on-screen buttons are still out of
scope.

## Nation sidebar

Stub built: `.sidebar` in `web/templates/game.html`/`game.css`, listing
England and France (hardcoded — matches this game's actual two-nation
map, not derived from `gamemap.GameMap.Nations`) with a color swatch each.
Visual direction settled as part of the stub: 1910s, "Toy Soldiers" —
warm wood/brass chrome (`--wood`/`--brass`/`--ivory`), one webfont (Alfa
Slab One, headers only — this project's first external dependency, scoped
tightly on purpose) over the system serif stack. Bold/saturated colors
rather than the classic muted palette: England `#1e3a6e` (navy), France
`#4a90d9` (bright royal blue, deliberately lighter than England's navy so
the two read as distinct).

Not yet done: wired to real data, hovering for stats/messaging. Messaging
itself isn't built yet — that part stays a placeholder/disabled affordance
until it exists, not something to fake.

Open question, still open: are these bold stub colors what ships, or does
the full 7-power roster get the classic muted palette instead (Austria
red, Germany black/grey, Italy green, Russia white, Turkey yellow)? Not
decided — only 2 of 7 nations have a real answer right now.

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

Stub built: second section of `.sidebar`, listing France's two real
starting units (`A Paris`, `F Brest` — from
`core/gamemap/testdata/western_europe.json`'s `starting_units`, reused as-
is since they're already correct) each with a made-up order ("Hold",
"Move → English Channel") to demonstrate the order slot exists.

Not yet done: wired to real data or real orders. Data:
`core/game.Unit{ID,NationID,ProvinceID,Type,Coast,DislodgedFrom}` already
has everything needed for the unit side of this; orders don't exist yet
(see "Selection & order building" below). Open question: does this list
show only your own units, or everyone's (already visible on the map
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
