# Map Notation

## Overview

The Diplomacy map is modeled around two related concepts:

- **Province** — an occupiable space on the board
- **Coast** — a fleet movement location attached to a fleet-accessible province

Every unit occupies a **province**. Fleets may also have a **coast** when coast-specific movement matters.

This distinction exists because armies and fleets use different movement graphs.

---

## Provinces

A province is any named space that can contain a unit.

There are three province types:

| Type | Can army occupy? | Can fleet occupy? | Example |
|---|---:|---:|---|
| `inland` | Yes | No | Paris |
| `coastal` | Yes | Yes | Brest |
| `water` | No | Yes | Mid-Atlantic Ocean |

The game state tracks unit positions by province:

```go
Positions map[gamemap.ProvinceID]UnitID
```

This answers the important board question: "which unit occupies this province?"

---

## Coasts

A coast is a node in the fleet movement graph.

Only fleet-accessible provinces have coasts:

| Province type | Coast behavior |
|---|---|
| `inland` | No coasts |
| `water` | One implicit coast, same ID as the province |
| `coastal` with one coast | One implicit coast, same ID as the province |
| `coastal` with multiple coasts | Multiple explicit coast IDs |

Examples:

```json
{ "id": "par", "type": "inland",  "coasts": [] }
{ "id": "mao", "type": "water",   "coasts": ["mao"] }
{ "id": "bre", "type": "coastal", "coasts": ["bre"] }
{ "id": "spa", "type": "coastal", "coasts": ["spa-nc", "spa-sc"] }
```

Water provinces still have a coast because fleets move coast-to-coast in the graph. A water province's only coast uses the same ID as the province because there is no ambiguity about which side of the water space the fleet occupies.

---

## Why Water Provinces Have Coasts

At first it can feel odd that a water province has a coast. The reason is implementation consistency.

Instead of special-casing water movement, all fleet movement uses one graph:

```go
FleetAdjacency map[gamemap.CoastID][]gamemap.CoastID
```

That means these are all the same kind of edge:

```text
bre -> mao
mao -> spa-nc
spa-sc -> por
eng -> lon
```

If water provinces did not have coast IDs, fleet movement would need two systems:

- coastal province coast-to-coast movement
- water province province-to-province movement

By giving water provinces a single implicit coast, all fleet movement becomes coast-to-coast.

---

## Army Movement vs Fleet Movement

Armies use province adjacency:

```go
ArmyAdjacency map[gamemap.ProvinceID][]gamemap.ProvinceID
```

Fleets use coast adjacency:

```go
FleetAdjacency map[gamemap.CoastID][]gamemap.CoastID
```

This matches the rules:

- Armies care about land province adjacency
- Fleets care about coastline/water connectivity
- Bicoastal provinces need explicit coast selection

Example:

```json
"army_adjacency": {
  "par": ["bre", "gas"],
  "gas": ["par", "bre", "spa"]
},
"fleet_adjacency": {
  "bre": ["eng", "mao", "gas"],
  "mao": ["eng", "bre", "gas", "spa-nc", "spa-sc", "por"],
  "spa-nc": ["gas", "mao", "por"],
  "spa-sc": ["mao", "por"]
}
```

---

## Bicoastal Provinces

Some coastal provinces have multiple coasts that do not connect to each other.

Example: Spain has a north coast and a south coast.

```json
{
  "id": "spa",
  "name": "Spain",
  "type": "coastal",
  "coasts": ["spa-nc", "spa-sc"]
}
```

A fleet entering Spain must specify which coast it enters:

```go
MoveOrder{
    Target:      "spa",
    TargetCoast: "spa-nc",
}
```

A fleet on `spa-nc` and a fleet on `spa-sc` are both in the province `spa`, but their legal fleet moves differ.

This is why game state uses both:

```go
Positions   map[gamemap.ProvinceID]UnitID
FleetCoasts map[UnitID]gamemap.CoastID
```

`Positions` says a fleet occupies Spain. `FleetCoasts` says which coast of Spain the fleet is on.

---

## Game State Position Rules

Every unit on the board appears in `Positions`:

```go
Positions["spa"] = "fra-fleet-spa-1901"
```

Every fleet also appears in `FleetCoasts`:

```go
FleetCoasts["fra-fleet-spa-1901"] = "spa-nc"
```

Armies do not appear in `FleetCoasts`.

For water provinces and single-coast coastal provinces, the fleet's coast usually matches the province ID:

```go
Positions["mao"] = "fra-fleet-mao-1901"
FleetCoasts["fra-fleet-mao-1901"] = "mao"
```

For bicoastal provinces, the fleet coast differs from the province ID:

```go
Positions["spa"] = "fra-fleet-spa-1901"
FleetCoasts["fra-fleet-spa-1901"] = "spa-nc"
```

---

## Order Validation Implications

Army movement validates against province adjacency:

```go
gm.CanArmyMove(fromProvince, targetProvince)
```

Fleet movement validates against coast adjacency:

```go
gm.CanFleetMove(sourceCoast, targetCoast)
```

For fleet orders:

- source coast comes from `Game.FleetCoasts[unitID]`
- target province comes from the order
- target coast may be inferred if the target has exactly one coast
- target coast is required if the target has multiple coasts

For army orders:

- target coast must be empty
- armies cannot enter water provinces

---

## Naming Convention

Use short lowercase IDs matching common Diplomacy abbreviations where possible.

Province IDs:

```text
par  Paris
bre  Brest
mao  Mid-Atlantic Ocean
spa  Spain
```

Coast IDs:

```text
bre     Brest's only coast
mao     Mid-Atlantic Ocean's only coast
spa-nc  Spain north coast
spa-sc  Spain south coast
```

A single-coast fleet-accessible province uses the same ID for its province and coast. A bicoastal province uses separate coast IDs.
