# User Experience & Frontend Architecture

## Purpose

This document captures decisions and direction for the player-facing UI. It
is a living document, same spirit as `docs/architecture.md` — write down
decisions as they're made, flag what's still open, and let it evolve as the
UI actually gets built. `docs/architecture.md` owns the domain/application
layering; this document owns everything from the Web layer outward: how the
frontend is built, how it talks to the backend, and how state reaches
players in real time.

## Two-flow architecture

The application has two flows, shaped differently enough that they get
different treatment rather than one framework stretched over both:

- **Account, lobby, and game-management**: home page, signup, login,
  browsing/creating games, inviting players, accepting invites. This is
  CRUD/document-shaped — load some rows, render them, handle a form POST.
  It doesn't need client-side routing or a global app-state store, and an
  invite link needs to open cold (from an email, a text message) without
  hydrating a JS bundle first to be useful.
- **In-game**: the board, orders, live state sync once you're actually
  playing a turn. Genuinely one continuously-running client application.

Decision: **Go renders the first flow directly** (`html/template`, stdlib —
real Go request-handling and template practice, not templating-library
archaeology), and **Svelte is scoped to the second flow only**. A Go handler
for `/games/{id}` renders a thin HTML shell — a mount div plus the game ID
(and maybe minimal initial state) embedded as a data attribute or inline
JSON — and that's where the Svelte app boots. One cookie session
authenticates both halves; there's no token handoff between them.

This keeps the Svelte bundle scoped to what gameplay actually needs (map,
order UI, WebSocket client) rather than re-implementing "table of my games"
in a client framework for no real gain, and it means the account/lobby side
gets real Go practice (handlers, templates, form handling, cookie auth)
instead of being routed through the same SPA machinery the game screen
needs for different reasons.

## In-game frontend (Svelte SPA)

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
    well; it doesn't fit this. (It's also why the account/lobby flow above
    is plain Go-rendered pages rather than HTMX — that flow *is* CRUD-shaped,
    but stdlib `html/template` covers it with no extra dependency at all.)
  - **Not React**, by choice: the goal is deliberately doing the UI
    differently from what's already familiar, and Svelte's compile-away
    runtime is a genuinely lighter-weight choice for a small, mostly
    client-rendered app like this one.
- Lives as its own project (proposed: a top-level `web/` directory alongside
  `cmd/`, `core/`, `application/`), talking to the Go backend over HTTP
  during development via a dev-server proxy, and served as static files in
  production.

## Account, lobby, and game-management pages

- Plain Go, `html/template` (stdlib), served by the same process as the
  in-game API. No Node, no build step, no separate origin — normal
  server-rendered pages with normal cookie-session auth and normal form
  POSTs.
- Covers: home/landing, signup, login, "my games," creating a game,
  inviting players, accepting/declining an invite.
- This is also where account credentials and sessions live once the
  authentication approach is settled (see Open Questions) — this flow is
  the natural home for login/logout handlers regardless of which credential
  model is chosen.

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

## Styling

Split the same way the frontend itself is split — each half's styling needs
are genuinely different:

- **Account/lobby pages: Pico.css.** A classless CSS framework — plain
  semantic HTML (`<button>`, `<nav>`, `<table>`, `<form>`), the stylesheet
  styles bare elements, no utility classes to learn or maintain. Fits this
  flow well specifically because those pages don't need custom design, and
  it costs nothing beyond a `<link>` tag — no build step, no PostCSS
  pipeline, consistent with that flow having no Node toolchain at all.
- **In-game SPA: Svelte's native scoped `<style>` blocks, plain modern CSS.**
  Every Svelte component's styles are automatically scoped to that
  component at compile time (compiled to unique class names) — CSS-Modules-
  style encapsulation with no extra tooling and no naming-convention
  discipline required. That removes the traditional argument against
  hand-written CSS at scale (global namespace collisions), and the map/order
  UI needs real custom layout no utility framework would meaningfully help
  with anyway. Modern CSS (native nesting, `:has()`, custom properties) is
  also just more capable than it used to be. Open Props (CSS custom
  properties for spacing/color/shadow scales, not utility classes) is worth
  a look later if a curated design-token system is wanted without going
  full utility-class — not decided, not blocking anything.
- Explicitly not Tailwind, by choice — it would likely get to a polished
  look fastest, but the goal here includes trying something other than the
  utility-class approach, and the bespoke game UI needs hand-written CSS
  regardless of what's chosen for the rest.

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

## Database

- **Postgres**, run locally via Podman — a standard `compose.yaml`
  (Podman's compose tooling reads the same format Docker's does, no
  Podman-specific dialect) with a Postgres service, a named volume, and a
  healthcheck.
- **Driver: `jackc/pgx`, used natively (`pgxpool.Pool`), not through
  `database/sql`.** `lib/pq` is community-legacy at this point; pgx is where
  the ecosystem's gone. Going through `database/sql`'s generic interface
  would buy portability to a different database this project has no
  intention of using, at the cost of pgx's native pooling and Postgres-
  specific features (JSONB, and `LISTEN`/`NOTIFY` as a possible future piece
  of the live-update story).
- **Query layer: `sqlc`.** Write real SQL, get generated type-safe Go
  scanning/binding code around it. Chosen over an ORM (GORM et al. abstract
  away exactly the SQL understanding worth building here, for a domain
  that's a handful of tables) and over raw hand-rolled `pgx` calls (sqlc
  removes the tedious boilerplate without hiding the SQL itself). Hand-
  rolled pgx queries remain a fine alternative if more manual practice is
  wanted for a given piece.
- **Hybrid schema**, following the shape `docs/architecture.md` already
  implies with its `UPDATE games SET state = ?, version = version + 1 ...`
  sketch:
  - Real relational tables for anything that needs to be queried or joined:
    `players` (credentials), `sessions`, `games` (id, map_id, status,
    created_at — enough metadata to list/filter without touching the
    blob), and `invites`/`assignments` (player↔game↔nation,
    pending/accepted/declined).
  - One JSONB column holding the serialized `*game.Game` aggregate for the
    actual turn-by-turn engine state. Nothing needs to SQL-query into
    individual units or orders — the aggregate always loads and saves
    atomically, matching the optimistic-concurrency version check already
    designed in `docs/architecture.md`.
- **Migrations**: `golang-migrate` or `goose` — both plain-SQL-file based,
  simple up/down migrations. Not yet chosen between the two; either is
  fine.

## Email

- **Mailpit** for local dev — the direct equivalent of Rails' `letter_opener`
  or Phoenix/Swoosh's local adapter: a fake SMTP server (actively
  maintained, written in Go, MailHog's successor) that captures everything
  sent to it and serves a web UI to view the rendered emails. Added as
  another service in the same Podman compose file used for Postgres.
- The app uses the *same* SMTP-sending code path in dev and production,
  just pointed at a different host — no test-only delivery method
  diverging from the real one, arguably a cleaner property than swapping
  adapters between environments.
- **`Mailer`/`InviteSender` interface in the application layer**, mirroring
  the existing `GameRepository`/`PlayerRepository` pattern: a real SMTP
  implementation (Mailpit locally, a real provider — Postmark/SES/etc. —
  in production, eventually) and a trivial in-memory fake for automated
  tests (append sent messages to a slice, assert against them — the same
  shape as `ActionMailer::Base.deliveries` in tests).
- Client library: `wneessen/go-mail` (modern, actively maintained) over
  stdlib `net/smtp`, which is usable for simple mail but notoriously
  low-level.

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

- Exact REST resource/endpoint design for the in-game API — not yet decided.
- WebSocket message vocabulary/protocol — not yet decided; likely derived
  from the event list above once it's needed.
- **Authentication: discussed, not concluded.** Leaning toward a server-side
  session with an opaque cookie (matches `Player`/`Game`/`Map` repository
  pattern already in this codebase — a `SessionRepository`, in-memory now,
  swappable later; also the practical choice for the live-update channel,
  since a browser attaches cookies to a WebSocket handshake automatically
  but won't attach arbitrary headers to one, so a JWT-in-header approach
  doesn't get that for free). Credential model itself — self-hosted
  username/password, OAuth, or something else — is still open.
- Whether the real-time channel is scoped per-game (subscribe to one game's
  updates) or per-player (a player's connection spans however many games
  they're in) — affects connection/subscription design, not yet decided.
- Migration tool: `golang-migrate` vs. `goose` — not yet chosen.

## Status

Nothing implemented yet. This document reflects direction agreed on before
any UI code exists, and should be updated as decisions firm up or change
during actual implementation.
