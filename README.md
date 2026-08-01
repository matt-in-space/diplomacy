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
                and repositories, currently in-memory only
cmd/
  server/       minimal process wiring
docs/           design docs and the project roadmap
```

`core` packages depend only on each other and never on `application`.
`adjudicator` takes a `*game.Game` and a `*gamemap.GameMap` and returns a
`Resolution` — it doesn't mutate anything. `game.Game` applies a `Resolution`
to itself.

## Getting started

```sh
go build ./...
go test ./...
```

There's no web layer yet, so `cmd/server` just wires the service together and
exits. Because it loads its fixture map with a path relative to the working
directory, run it from inside `cmd/server/`, not the repo root:

```sh
cd cmd/server && go run .
```

## Docs

Start with [`docs/roadmap.md`](docs/roadmap.md) for current status. The rest
of `docs/` covers specific design areas (map notation, order flow, the
adjudication resolver, context propagation, testing conventions) — some of it
predates the current implementation and is flagged as stale in the roadmap
rather than kept in sync inline.
