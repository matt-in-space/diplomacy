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
frontend/       TypeScript source for the in-game screen — compiled by
                `tsc` (no bundler) to web/static/js/, loaded as native
                browser ES modules from web/templates/game.html
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

Running the server needs the TypeScript frontend compiled first —
`web/static/js/` is empty until `npm run build` has populated it, and
`go:embed` bundles whatever's on disk at build time, so skipping this just
means the game screen serves a 404 for its script. Easiest path, via
[`mise`](https://mise.jdx.dev) (pins Go 1.26.4 and Node 22 — see
`mise.toml`):

```sh
mise trust   # first time only, in a fresh clone
mise run dev # installs frontend deps, builds it, then runs the seeded dev server
```

`mise run dev` skips the install/build steps on later runs if nothing under
`frontend/` changed. Without `mise`, the same thing by hand:

```sh
cd frontend && npm install && npm run build
cd ..
go run ./cmd/server            # add -seed to create dev accounts
                                # (user1@example.com / user2@example.com, password: password)
```

There's no live-reload for either side yet: editing a Go template or the
compiled frontend both require a rebuild to see the change, same as any
other `go:embed`'d file. For the frontend specifically, `npm run watch`
(in `frontend/`) recompiles on save — you'd still restart the server to
pick the new output up.

## Docs

Start with [`docs/roadmap.md`](docs/roadmap.md) for current status. The rest
of `docs/` covers specific design areas (map notation, order flow, the
adjudication resolver, context propagation, testing conventions) — some of it
predates the current implementation and is flagged as stale in the roadmap
rather than kept in sync inline.
