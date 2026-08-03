# User Experience & Frontend Architecture

## Purpose

This document captures decisions and direction for the player-facing UI. It
is a living document, same spirit as `docs/architecture.md` — write down
decisions as they're made, flag what's still open, and let it evolve as the
UI actually gets built. `docs/architecture.md` owns the domain/application
layering; this document owns everything from the Web layer outward: how the
frontend is built, how it talks to the backend, and how state reaches
players in real time.

## Frontend

- **Svelte, not SvelteKit.** A plain Svelte SPA built with Vite, compiled to
  static assets. SvelteKit brings its own Node server (SSR, routing, API
  routes) that would duplicate the role the Go application layer already
  owns — this project wants one backend, not two.
- Chosen deliberately over both HTMX and React:
  - **Not HTMX.** The interaction shape here — select a unit, see legal
    destinations highlight, build up an order, hold several drafted orders
    client-side before committing, hover previews — is real transient
    client-side state that shouldn't round-trip to the server per
    interaction. HTMX fits CRUD-shaped apps (forms, lists, partial swaps)
    well; it doesn't fit this.
  - **Not React**, by choice: the goal is deliberately doing the UI
    differently from what's already familiar, and Svelte's compile-away
    runtime is a genuinely lighter-weight choice for a small, mostly
    client-rendered app like this one.
- Lives as its own project (proposed: a top-level `web/` directory alongside
  `cmd/`, `core/`, `application/`), talking to the Go backend over HTTP
  during development via a dev-server proxy, and served as static files in
  production.

## Map rendering

- **SVG, not Canvas.** Decided and not revisited: a Diplomacy board is ~34
  fixed provinces and at most ~34 units, turn-based, no continuous animation
  loop — none of the conditions that make Canvas worth its cost.
- Reasoning:
  - Hit-testing is free: each province is a `<path>` with its own
    click/hover handlers, versus manual point-in-polygon math against raw
    coordinates on Canvas.
  - Styling (highlight legal destinations, show ownership color, mark a
    selected unit) is reactive attribute/class binding on real DOM nodes,
    not manual redraw logic.
  - SVG elements are DOM elements, so Svelte's reactivity applies directly —
    `{#each provinces as p}<path fill={colorFor(p)} onclick={...}/>{/each}`
    is idiomatic Svelte with no adapter layer. Canvas drawing is imperative
    and would fight the framework instead of using it.
  - The classic Diplomacy board already exists as a labeled SVG map with
    named regions — real reuse potential, not just a theoretical advantage.
  - If move-animation polish is wanted later (units sliding to their new
    province, arrows drawing in), CSS transitions/SMIL on SVG nodes handle
    that without needing Canvas.
- One rendering system, not two: the map is Svelte-rendered SVG, part of the
  same component tree as the surrounding UI chrome — not a separate canvas
  layer coordinated underneath it.

## API shape

- **RESTful by default.** The core engine's shape — a `Game` aggregate,
  commands like submit-order/commit/advance, phase-scoped resources — maps
  reasonably well onto REST resources and this is the default style.
- **Not beholden to it.** Where REST is a poor fit — live state updates —
  use the right tool instead of forcing it through polling or contorting
  the resource model. See Live Updates below.
- The API should return domain-computed legal-move data (`LegalRetreats`,
  `LegalBuildProvinces`, `AdjustmentBalance`, etc.) rather than making the
  frontend reimplement any of it in JS. One authoritative rules
  implementation stays in `core/game`/`core/adjudicator`; the frontend only
  ever renders what the server already decided was legal.

## Live updates

- **Decision: build this properly from the start, not deferred.** Turn
  timing is expected to vary a lot by game — a relaxed game giving players
  days to collect orders doesn't need live updates, but a fast game with,
  say, 15-minute movement/diplomacy windows genuinely does; players need to
  see orders committing and phases advancing without refreshing. Rather than
  ship a polling-based v1 and rebuild for real-time later, invest in it now
  — it's also a deliberate Go-learning goal for this project.
- Mechanism: a real-time channel (WebSocket, or SSE where the direction is
  purely server-to-client) supplements the REST API rather than replacing
  it. Commands (submit an order, commit, etc.) stay REST requests; board-
  state change notifications flow over the real-time channel to connected
  clients.
- This connects naturally to `docs/architecture.md`'s event vocabulary
  (`OrdersRequested`, `OrdersResolved`, `RetreatsRequested`, `PhaseAdvanced`,
  `GameCompleted`) — those are the natural set of things to push to
  connected clients once that event/outbox machinery exists. Until then, a
  simpler in-process broadcast from the phase-processing path can serve the
  same purpose without needing the durable outbox that document describes
  for production-grade multiplayer resilience.

## Relationship to `docs/architecture.md`

No conflict — the Web layer described there is already framework- and
format-agnostic ("adapts HTTP requests to application commands... translates
results into HTTP responses"). JSON-over-HTTP plus a WebSocket channel is one
valid implementation of that layer, not a departure from it.

Sequencing matters, though: that document's "Initial Implementation
Sequence" defers durable jobs and the transactional outbox to step 5, after
a complete synchronous order-submission path exists. UI work should follow
the same order — build against simple synchronous handlers first (submit an
order, get an immediate response), and only reach for outbox/job durability
once there's something actually playable to point the UI at. Building the
production-multiplayer job infrastructure before there's a UI to exercise it
would be a long detour from getting the game on screen.

## Open questions

- Exact REST resource/endpoint design — not yet decided.
- WebSocket message vocabulary/protocol — not yet decided; likely derived
  from the event list above once it's needed.
- Authentication/session model for players — not yet discussed.
- Whether the real-time channel is scoped per-game (subscribe to one game's
  updates) or per-player (a player's connection spans however many games
  they're in) — affects connection/subscription design, not yet decided.

## Status

Nothing implemented yet. This document reflects direction agreed on before
any UI code exists, and should be updated as decisions firm up or change
during actual implementation.
