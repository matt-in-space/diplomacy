# Diplomacy — Architecture

## Overview

This document covers the technical architecture of the Diplomacy web application built in Go. It is a living document and will evolve as decisions are made.

The application has two major surfaces:
- **Game Engine** — pure domain logic: state machine, map graph, order validation, adjudication
- **Web Application** — HTTP API, persistence, background processing, notifications

---

## Project Structure

```
diplomacy/
├── cmd/
│   └── server/
│       └── main.go          # composition root: wires all layers, starts HTTP server + worker
├── internal/
│   ├── game/                # pure domain — zero infrastructure dependencies
│   │   ├── game.go          # Game struct, state machine, PutOrder, Advance
│   │   ├── order.go         # Order interface + concrete types (Move, Hold, Support, Convoy)
│   │   └── adjudicator/     # Resolve() — pure function, orders in → resolution out
│   ├── gamemap/             # province graph, coast adjacency, legal move queries
│   ├── commands/            # application service layer — orchestrates domain + infrastructure
│   │   ├── submit_order.go
│   │   ├── advance_game.go
│   │   └── ...
│   ├── store/               # persistence — repository interfaces + Postgres implementations
│   │   └── game_repo.go
│   ├── worker/              # background job loop + cron deadline checker
│   ├── notify/              # email + in-app notification implementations
│   └── api/                 # HTTP handlers, routing, middleware
├── docs/
└── go.mod
```

### Dependency Direction

Outer layers depend on inner. The `game` package never imports `store`, `api`, `notify`, or `worker`.

```
api / worker
    ↓
commands
    ↓
game (domain)
    ↑
store (implements interfaces defined in commands)
```

`cmd/server/main.go` is the only place where all layers are wired together.

---

## Core Systems

### 1. Game Domain (`internal/game`)

The `Game` struct is the aggregate root. It holds all authoritative game state and exposes methods that delegate to isolated domain packages. It has no knowledge of HTTP, Postgres, or job queues.

**State held by `Game`:**
- Current phase and year
- Unit positions (province → unit)
- Supply center ownership
- Submitted orders (player → order set)
- Pending retreats (after resolution)
- Player assignments

**Key methods:**
- `game.PutOrder(playerID, order) error` — validates and stores an order; rejects illegal orders immediately
- `game.AllOrdersSubmitted() bool` — used by the command layer to decide whether to enqueue an advance job
- `game.Advance()` — transitions the state machine to the next phase; calls the adjudicator if in the order resolution phase
- `game.LockOrders(playerID) error` — marks a player's orders as final

---

### 2. Map Graph (`internal/gamemap`)

The map is not a generic graph struct with vertex/edge objects — it is an adjacency list backed by maps. The only queries Diplomacy requires are "is X adjacent to Y" and "what are the neighbors of X", both O(1) with this approach. No pathfinding or traversal abstraction is needed.

```go
type Map struct {
    Provinces      map[ProvinceID]Province
    ArmyAdjacency  map[ProvinceID][]ProvinceID
    FleetAdjacency map[CoastID][]CoastID
}
```

#### Province Types

There are three province types: `Inland`, `Water`, and `Coastal`.

- **Water provinces** are full provinces (e.g. North Sea, Mediterranean). Fleets occupy and move through them. They never appear in army adjacency. They have a single implicit coast with the same ID as the province.
- **Inland provinces** appear only in army adjacency. Fleets cannot enter them.
- **Coastal provinces** appear in both army adjacency (by province ID) and fleet adjacency (by coast ID). Most have a single coast; bicoastal provinces (Spain, Bulgaria, St. Petersburg) have two named coasts.

#### Coast Model

Every fleet-navigable province has at least one coast entry, even if implicit. This makes fleet adjacency uniformly coast-to-coast across all cases:

| Province type | Coasts |
|---|---|
| Water | One coast, ID = province ID (e.g. `"nth"`) |
| Single-coast coastal | One coast, ID = province ID (e.g. `"bre"`) |
| Bicoastal coastal | Two named coasts (e.g. `"spa-nc"`, `"spa-sc"`) |

The rule "armies cannot enter water" is implicit in the data — water provinces simply have no army adjacency entries.

#### Embedded JSON

Map data is stored as a JSON file embedded into the binary at compile time using Go's `//go:embed` directive. This means no file path dependencies at runtime and no separate config to deploy. Swapping maps (dev subset → full map → custom map) is a data-only change.

```go
//go:embed data.json
var rawMapData []byte

func Load() (*Map, error) {
    var data mapFileData
    if err := json.Unmarshal(rawMapData, &data); err != nil {
        return nil, err
    }
    return buildMap(data), nil
}
```

`buildMap()` converts the raw deserialized shape into the runtime struct with its lookup maps, keeping the JSON schema decoupled from the in-memory representation.

**JSON structure:**

```json
{
  "id": "standard",
  "name": "Classic Diplomacy",
  "victory_threshold": 18,
  "nations": ["England", "France", "Germany", "Italy", "Austria", "Russia", "Turkey"],
  "provinces": [
    { "id": "nth", "name": "North Sea",  "type": "water",   "supply_center": false, "home_nation": null,     "coasts": ["nth"] },
    { "id": "bre", "name": "Brest",      "type": "coastal", "supply_center": true,  "home_nation": "France", "coasts": ["bre"] },
    { "id": "spa", "name": "Spain",      "type": "coastal", "supply_center": true,  "home_nation": null,     "coasts": ["spa-nc", "spa-sc"] },
    { "id": "par", "name": "Paris",      "type": "inland",  "supply_center": true,  "home_nation": "France", "coasts": [] }
  ],
  "army_adjacency": {
    "par": ["pic", "bur", "gas", "bre"],
    "spa": ["por", "gas", "mar"]
  },
  "fleet_adjacency": {
    "nth": ["nwg", "eng", "hel", "ska", "edi", "yor"],
    "bre": ["eng", "mao", "pic", "gas"],
    "spa-nc": ["gas", "mao", "por"],
    "spa-sc": ["wmd", "gol", "por", "mar"]
  }
}
```

Nations, victory threshold, and starting positions are all map-defined, not hardcoded — this supports custom maps (Westeros, Middle Earth, etc.) as alternative datasets with no engine changes.

**Key queries the map exposes:**
- `CanMove(unitType, from, to) bool`
- `LegalMoves(unitType, from) []ProvinceID`
- `IsCoastal(provinceID) bool`
- `CostsFor(provinceID) []CoastID`

Convoy route validation (does a chain of fleets connect two coastal provinces?) is a BFS over `FleetAdjacency` — a single function, not a reason to build a general traversal abstraction.

---

### 3. Order Types (`internal/game/order.go`)

Orders use an interface with concrete types. This makes the adjudicator's type switches explicit and exhaustive, and keeps each order's fields minimal and typed.

```go
type Order interface {
    UnitID() string
    Validate(state *GameState, m *gamemap.Map) error
}

type HoldOrder struct {
    Unit string
}

type MoveOrder struct {
    Unit string
    From string
    To   string
}

type SupportOrder struct {
    Unit          string
    SupportedUnit string
    TargetProvince string  // province being supported into (for move support)
}

type ConvoyOrder struct {
    Fleet string
    Army  string
    From  string
    To    string
}
```

Serialization to/from JSON for Postgres storage is handled at the `store` layer, not in the domain types themselves.

---

### 4. Adjudicator (`internal/game/adjudicator`)

A pure, stateless function. Takes a snapshot of game state and a complete order set, returns a resolution result. Has no side effects.

```go
func Resolve(state GameState, orders []Order) Resolution
```

**`Resolution` contains:**
- Which units moved successfully
- Which units were dislodged (and from where)
- Which orders failed and why
- Provinces vacated by standoff (used in retreat phase to restrict valid retreat destinations)

#### Resolution Algorithm

1. **Validation pass** — illegal orders (e.g. army moving to water) become Holds silently
2. **Compute raw support** — for each Move, count valid SupportOrders targeting it; strength = 1 + support count
3. **Cut support** — a Support is cut if the supporting unit is attacked from any province other than the one it is supporting into, by a foreign unit; iterate until stable
4. **Resolve moves:**
   - **Bounces** — two moves into the same province with equal strength: both fail
   - **Dislodgement** — attacker strength > defender strength (hold + defensive support)
   - **Circular moves** — detect cycles; all units in a cycle succeed together if no external blocker
   - **Chain blockage** — if the head of a chain bounces, failure propagates back through the chain
5. **Dislodged unit attacks** — a dislodged unit still executes its attack order unless it was attacking the province that dislodged it
6. **Build retreat state** — collect dislodged units, record attacker origin and standoff provinces for retreat validation

The adjudicator is the primary target for DATC test cases. Because it is a pure function it requires no game setup to test.

---

### 5. State Machine (`internal/game/game.go`)

Phases as an explicit enum. `game.Advance()` drives transitions.

```
Lobby
  → SpringDiplomacy
  → SpringOrders
  → SpringResolution      ← adjudicator runs here
  → SpringRetreats
  → FallDiplomacy
  → FallOrders
  → FallResolution        ← adjudicator runs here
  → FallRetreats
  → Adjustments           ← build/disband orders; unit count reconciled
  → SpringDiplomacy (next year)
```

`Diplomacy` phases have no mechanical engine effect — they are timer windows for player communication. The engine waits for the deadline or an explicit trigger.

---

### 6. Application Service Layer (`internal/commands`)

Command handlers sit between the HTTP/worker layer and the domain. They accept infrastructure interfaces (not concrete types) so they remain independently testable.

**Example — submitting an order:**
```
SubmitOrderCommand { GameID, PlayerID, Order }
  → load game from repo
  → game.PutOrder(playerID, order)
  → gameRepo.Save(game)
  → if game.AllOrdersSubmitted(): jobQueue.Enqueue(AdvanceGameJob{GameID})
  → return
```

**Example — advancing a game (called by worker):**
```
AdvanceGameJob { GameID }
  → load game from repo
  → game.Advance()           // adjudicates internally if in resolution phase
  → gameRepo.Save(game)
  → notifier.Notify(game.ConsumeEvents())
```

The `Game` struct accumulates domain events (e.g. `PhaseAdvanced`, `UnitDislodged`, `GameOver`) during `Advance()` which the command handler drains and passes to the notifier.

---

### 7. Background Worker & Cron (`internal/worker`)

Two triggers cause a game to advance:
1. **All players lock orders** — controller enqueues `AdvanceGameJob` after the last lock
2. **Deadline expires** — cron polls for games past their deadline and enqueues them

Both paths enqueue the same job, processed by the same worker. The worker holds a DB-level advisory lock per game before calling `Advance()` to prevent double-resolution if both triggers fire simultaneously.

---

## Key Design Decisions

| Decision | Choice | Reason |
|---|---|---|
| Order types | Interface + concrete structs | Exhaustive type switches, minimal fields per type, testable |
| Map data | Embedded JSON, sibling `gamemap` package | Data/logic separation, easy to swap maps, avoids import cycles |
| Custom map support | Nations/threshold/positions in JSON | Engine has no hardcoded assumptions about standard Diplomacy map |
| Adjudicator | Pure function, separate package | Fully testable against DATC cases without game setup |
| Advance trigger | Background worker + job queue | Single code path for deadline and early resolution; HTTP layer stays thin |
| State serialization | JSON column in Postgres | Simple for a single aggregate; revisit if query patterns demand normalization |
| Bicoastal provinces | Coast-level graph edges | No special cases needed in adjudicator or validation |

---

## Open Questions

- What does the full retreat order submission flow look like (deadline, auto-disband rules)?
- How are adjustment (build/disband) orders structured — same `Order` interface or separate?
- What is the dev subset map? A small slice of the real map or a purpose-built test map?
- Where do player communication/messaging features live — in-app only, or does the engine need to model the Diplomacy phase at all?
