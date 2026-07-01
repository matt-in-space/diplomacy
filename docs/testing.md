# Diplomacy — Testing

## Overview

Testing should prioritize the game engine internals first: map loading/querying, order validation, adjudication, and state transitions. The web/API layer can be tested later through command handlers and HTTP integration tests.

The most important rule is to keep domain tests small, deterministic, and easy to read. Diplomacy adjudication has many edge cases, so test setup should not obscure the board state being tested.

---

## Test Helper Organization

Domain-specific test helpers should live near the domain package they support. Generic helpers can live in a top-level test utility package if needed later.

Recommended structure:

```
internal/
└── game/
    ├── gamemap/
    │   ├── gamemap.go
    │   └── maptest/
    │       └── maptest.go      # shared map test builders
    ├── adjudicator/
    └── game.go
```

Use `internal/game/gamemap/maptest` for helpers that build `gamemap.GameMap` values.

Avoid starting with a broad `internal/testutil` package. It can easily become a dumping ground. Use top-level test utilities only for truly generic helpers such as fake clocks, fake repositories, ID generators, or assertion helpers.

Rule of thumb:

- **Domain-specific test helpers** live beside their domain
- **Generic test helpers** live in a top-level utility package

---

## Why `maptest` Uses Normal `.go` Files

The shared map builder should be in a normal Go file, not a `_test.go` file:

```
internal/game/gamemap/maptest/maptest.go
```

This lets tests in multiple packages import it, such as:

- `internal/game/gamemap`
- `internal/game/adjudicator`
- `internal/game`

A package made only of `_test.go` files is not reusable as a normal imported package by other package tests in the way needed here.

This does **not** mean `maptest` becomes part of the production binary. It is only compiled into production if production code imports it. Since the package name is explicitly `maptest`, importing it from non-test code should be treated as a clear mistake.

For helpers used only inside one package's own tests, prefer a local `_test.go` helper file instead:

```
internal/game/gamemap/
├── gamemap.go
├── gamemap_test.go
└── helpers_test.go
```

---

## Builder Pattern for Map Tests

Go developers often avoid Java-style builder patterns in production code when a simple struct literal or constructor is enough. However, a small builder is appropriate for test setup when it reduces noise and keeps tests readable.

The map graph is a good candidate because a valid map requires multiple related structures:

- `Provinces`
- `ArmyAdjacency`
- `FleetAdjacency`
- implicit coast data for water and coastal provinces

A test builder helps avoid repetitive setup and reduces the chance of constructing inconsistent test maps.

The builder should stay limited to tests. The production API should remain simple structs, constructors, and methods.

---

## Example Map Builder Shape

Example package:

```go
package maptest

import "diplomacy/internal/game/gamemap"

type Builder struct {
    m *gamemap.GameMap
}

func New() *Builder {
    return &Builder{
        m: &gamemap.GameMap{
            Provinces:      make(map[gamemap.ProvinceID]gamemap.Province),
            ArmyAdjacency:  make(map[gamemap.ProvinceID][]gamemap.ProvinceID),
            FleetAdjacency: make(map[gamemap.CoastID][]gamemap.CoastID),
        },
    }
}

func (b *Builder) Inland(id, name string) *Builder {
    pid := gamemap.ProvinceID(id)
    b.m.Provinces[pid] = gamemap.Province{
        ID:   pid,
        Name: name,
        Type: gamemap.Inland,
    }
    return b
}

func (b *Builder) Water(id, name string) *Builder {
    pid := gamemap.ProvinceID(id)
    cid := gamemap.CoastID(id)
    b.m.Provinces[pid] = gamemap.Province{
        ID:     pid,
        Name:   name,
        Type:   gamemap.Water,
        Coasts: []gamemap.CoastID{cid},
    }
    return b
}

func (b *Builder) Coastal(id, name string) *Builder {
    pid := gamemap.ProvinceID(id)
    cid := gamemap.CoastID(id)
    b.m.Provinces[pid] = gamemap.Province{
        ID:     pid,
        Name:   name,
        Type:   gamemap.Coastal,
        Coasts: []gamemap.CoastID{cid},
    }
    return b
}

func (b *Builder) Bicoastal(id, name string, coasts ...string) *Builder {
    pid := gamemap.ProvinceID(id)
    coastIDs := make([]gamemap.CoastID, 0, len(coasts))
    for _, coast := range coasts {
        coastIDs = append(coastIDs, gamemap.CoastID(coast))
    }
    b.m.Provinces[pid] = gamemap.Province{
        ID:     pid,
        Name:   name,
        Type:   gamemap.Coastal,
        Coasts: coastIDs,
    }
    return b
}

func (b *Builder) SupplyCenter(id, homeNation string) *Builder {
    pid := gamemap.ProvinceID(id)
    p := b.m.Provinces[pid]
    p.SupplyCenter = true
    p.HomeNation = homeNation
    b.m.Provinces[pid] = p
    return b
}

func (b *Builder) ArmyEdge(a, c string) *Builder {
    from := gamemap.ProvinceID(a)
    to := gamemap.ProvinceID(c)
    b.m.ArmyAdjacency[from] = append(b.m.ArmyAdjacency[from], to)
    b.m.ArmyAdjacency[to] = append(b.m.ArmyAdjacency[to], from)
    return b
}

func (b *Builder) FleetEdge(a, c string) *Builder {
    from := gamemap.CoastID(a)
    to := gamemap.CoastID(c)
    b.m.FleetAdjacency[from] = append(b.m.FleetAdjacency[from], to)
    b.m.FleetAdjacency[to] = append(b.m.FleetAdjacency[to], from)
    return b
}

func (b *Builder) Build() *gamemap.GameMap {
    return b.m
}
```

---

## Example Test Usage

```go
func TestArmyCanMoveBetweenAdjacentLandProvinces(t *testing.T) {
    m := maptest.New().
        Inland("par", "Paris").
        Coastal("bre", "Brest").
        ArmyEdge("par", "bre").
        Build()

    if !m.CanMove(gamemap.Army, "par", "bre") {
        t.Fatal("expected army to move from Paris to Brest")
    }
}
```

Example bicoastal setup:

```go
func TestFleetUsesSpecificCoast(t *testing.T) {
    m := maptest.New().
        Water("mao", "Mid-Atlantic Ocean").
        Water("wmd", "Western Mediterranean").
        Bicoastal("spa", "Spain", "spa-nc", "spa-sc").
        FleetEdge("mao", "spa-nc").
        FleetEdge("wmd", "spa-sc").
        Build()

    // A fleet in MAO can reach Spain's north coast, but not its south coast.
}
```

---

## Suggested Test Maps

Start with a small western Europe subset that covers all important map cases:

- `par` — Paris, inland, supply center, home France
- `bre` — Brest, coastal, supply center, home France
- `gas` — Gascony, coastal, non-supply center
- `mao` — Mid-Atlantic Ocean, water
- `eng` — English Channel, water
- `lon` — London, coastal, supply center, home England
- `spa` — Spain, bicoastal, supply center, neutral
- `por` — Portugal, coastal, supply center, neutral

This subset is enough to test:

- inland movement
- water movement
- coastal movement
- bicoastal fleet behavior
- supply centers
- home centers
- army/fleet movement differences
- basic convoy routes
- bounces and support-cutting scenarios

Example JSON fixture:

```json
{
  "id": "western-europe-subset",
  "name": "Western Europe Test Subset",
  "victory_threshold": 3,
  "nations": ["England", "France"],
  "provinces": [
    {
      "id": "par",
      "name": "Paris",
      "type": "inland",
      "supply_center": true,
      "home_nation": "France",
      "coasts": []
    },
    {
      "id": "bre",
      "name": "Brest",
      "type": "coastal",
      "supply_center": true,
      "home_nation": "France",
      "coasts": ["bre"]
    },
    {
      "id": "gas",
      "name": "Gascony",
      "type": "coastal",
      "supply_center": false,
      "home_nation": "",
      "coasts": ["gas"]
    },
    {
      "id": "mao",
      "name": "Mid-Atlantic Ocean",
      "type": "water",
      "supply_center": false,
      "home_nation": "",
      "coasts": ["mao"]
    },
    {
      "id": "eng",
      "name": "English Channel",
      "type": "water",
      "supply_center": false,
      "home_nation": "",
      "coasts": ["eng"]
    },
    {
      "id": "lon",
      "name": "London",
      "type": "coastal",
      "supply_center": true,
      "home_nation": "England",
      "coasts": ["lon"]
    },
    {
      "id": "spa",
      "name": "Spain",
      "type": "coastal",
      "supply_center": true,
      "home_nation": "",
      "coasts": ["spa-nc", "spa-sc"]
    },
    {
      "id": "por",
      "name": "Portugal",
      "type": "coastal",
      "supply_center": true,
      "home_nation": "",
      "coasts": ["por"]
    }
  ],
  "army_adjacency": {
    "par": ["bre", "gas"],
    "bre": ["par", "gas"],
    "gas": ["par", "bre", "spa"],
    "spa": ["gas", "por"],
    "por": ["spa"]
  },
  "fleet_adjacency": {
    "eng": ["lon", "bre", "mao"],
    "lon": ["eng"],
    "bre": ["eng", "mao", "gas"],
    "gas": ["bre", "mao", "spa-nc"],
    "mao": ["eng", "bre", "gas", "spa-nc", "spa-sc", "por"],
    "spa-nc": ["gas", "mao", "por"],
    "spa-sc": ["mao", "por"],
    "por": ["mao", "spa-nc", "spa-sc"]
  }
}
```

Notes:

- The adjacency lists are bidirectional and should include both directions explicitly.
- `lon` has no army adjacency in this subset because its real land neighbors are not included.
- `spa` is represented as one province with two fleet coasts: `spa-nc` and `spa-sc`.
- Water provinces (`mao`, `eng`) are still provinces, but only appear in `fleet_adjacency`.

---

## JSON Fixtures vs Code Builders

Both approaches are useful, but for different purposes.

Use **code builders** when:

- The test needs a tiny custom board
- The important setup should be visible directly in the test
- You want to avoid maintaining lots of fixture files

Use **JSON fixtures** when:

- Testing the map loader itself
- Testing a known reusable map subset
- Testing realistic full-map behavior
- Importing DATC-style scenarios later

The production map can still use embedded JSON, while most domain tests can use `maptest` builders directly.
