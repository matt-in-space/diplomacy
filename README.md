# Diplomacy

A Go implementation of the board game *Diplomacy*. This is a learning and
portfolio project — the priority is correct, idiomatic Go over feature
completeness. See [`docs/roadmap.md`](docs/roadmap.md) for what's implemented
and what isn't.

## Layout

```
core/
  gamemap/      static map data — provinces, coasts, army/fleet adjacency
  game/         the Game aggregate: units, orders, turns, phases, validation
  adjudicator/  resolves a turn's orders into a Resolution; no side effects
application/
  gameplay/     use-case orchestration (submit/commit orders, process phases)
  auth/         signup/login/session handling
  lobby/        game setup: create, join by code, start
  (all three above are repository interfaces + services, in-memory only for now)
infrastructure/
  memory/       in-memory repository implementations
web/            Go-rendered account/lobby pages (html/template) and the
                in-game route's HTML shell; static assets
  static/js/    hand-written ES modules (.mjs) for the in-game screen,
                loaded as native browser modules from
                web/templates/game.html — no build step, no bundler.
                Tests live alongside in static/js/_tests/, excluded from
                go:embed by their leading underscore.
cmd/
  server/       process wiring; `-seed` creates fixed dev accounts on startup
docs/           design docs and the project roadmap
```

`core` packages depend only on each other and never on `application`.
`adjudicator` takes a `*game.Game` and a `*gamemap.GameMap` and returns a
`Resolution` — it doesn't mutate anything. `game.Game` applies a
`Resolution` to itself.

## Getting started

```sh
go build ./...
go test ./...
```

The frontend is plain ES modules with no build step, so running the server
needs nothing beyond Go itself:

```sh
go run ./cmd/server            # add -seed to create dev accounts and a
                                # ready-to-start game (user1@example.com /
                                # user2@example.com, password: password —
                                # user1 hosting, user2 already joined; the
                                # server logs the lobby URL on startup)
```

Or via [`mise`](https://mise.jdx.dev) (pins Go 1.26.4 and Node 22 — Node is
only needed to run the frontend's test suite, see `mise.toml`):

```sh
mise trust   # first time only, in a fresh clone
mise run dev # runs the seeded dev server
```

There's no live-reload: editing a Go template requires a rebuild
(`go:embed` bundles whatever's on disk at build time), and editing a
`web/static/` file requires restarting the server for `go:embed` to pick
it up again. `go:embed` re-reads nothing at request time.

## Docs

Start with [`docs/roadmap.md`](docs/roadmap.md) for current status. The rest
of `docs/` covers specific design areas (map notation, order flow, the
adjudication resolver, context propagation, testing conventions) — some of it
predates the current implementation and is flagged as stale in the roadmap
rather than kept in sync inline.
