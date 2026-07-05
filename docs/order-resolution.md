# Order Resolution

## Overview

Order resolution, also called adjudication, determines the outcome of a submitted set of unit orders.

Diplomacy orders are resolved **simultaneously**. They should not be processed one at a time. A move may depend on support, support may be cut by an attack, and multiple units may contest the same province. Because of this, resolution should be implemented as a multi-pass analysis over the full order set.

The adjudicator should produce a **resolution result** first. The game state should apply that result afterward.

```go
resolution := adjudicator.Resolve(game, gameMap)
game.ApplyResolution(resolution)
```

This keeps resolution testable and prevents partially-mutated game state while adjudication is still being computed.

---

## Responsibilities

### Adjudicator

The adjudicator is responsible for answering:

- Which orders succeeded?
- Which orders failed?
- Which supports were cut?
- Which moves bounced?
- Which units moved?
- Which units were dislodged?
- Which units must retreat?
- Which provinces are unavailable for retreat?

The adjudicator should not:

- Save game state
- Send notifications
- Advance deadlines
- Know about players or UI state
- Mutate the game while resolving

### Game

The `Game` applies the adjudication result later:

- Move successful units
- Mark dislodged units
- Create retreat requirements
- Clear resolved orders
- Advance the turn
- Store the last resolution for user display

---

## Resolution Output

The first major artifact should be a `Resolution` struct.

Possible shape:

```go
type Resolution struct {
    Turn Turn

    OrderResults []OrderResult
    Moves        []MoveResult
    Dislodged    []DislodgedUnit
    Retreats     []RetreatRequired
    Standoffs    []Standoff
    CutSupports  []CutSupport
    FailedConvoys []FailedConvoy
}
```

The exact fields can evolve as adjudication is implemented.

---

## Order Results

Users need to see what happened to each order.

```go
type OrderResult struct {
    UnitID game.UnitID
    Order  game.Order
    Status OrderStatus
    Reason string
}
```

Possible statuses:

```go
type OrderStatus string

const (
    OrderSucceeded OrderStatus = "succeeded"
    OrderFailed    OrderStatus = "failed"
    OrderCut       OrderStatus = "cut"
    OrderInvalid   OrderStatus = "invalid"
)
```

Examples:

```text
A Paris -> Burgundy failed: bounced
A Gascony supports A Paris -> Burgundy failed: support cut
F Mid-Atlantic convoys A Brest -> London failed: convoy disrupted
```

---

## Move Results

Movement changes board state, so successful moves should be explicit.

```go
type MoveResult struct {
    UnitID game.UnitID
    From   gamemap.ProvinceID
    To     gamemap.ProvinceID

    FromCoast gamemap.CoastID
    ToCoast   gamemap.CoastID

    Success bool
    Reason  string
}
```

---

## Dislodgement and Retreats

A unit is dislodged when a stronger attack succeeds against its province.

```go
type DislodgedUnit struct {
    UnitID       game.UnitID
    Province     gamemap.ProvinceID
    DislodgedBy  game.UnitID
    AttackerFrom gamemap.ProvinceID
}
```

A retreat requirement is derived from dislodgement:

```go
type RetreatRequired struct {
    UnitID       game.UnitID
    From         gamemap.ProvinceID
    DislodgedBy  game.UnitID
    AttackerFrom gamemap.ProvinceID
    Options      []gamemap.ProvinceID
}
```

Retreat options exclude:

- occupied provinces
- the province from which the attacker came
- provinces left vacant due to a standoff during the same turn

Retreat order submission and retreat resolution are separate phases and should be implemented after base order adjudication.

---

## Internal Resolution Context

The adjudicator should reshape the game state into lookup-friendly maps before resolving.

```go
type resolutionContext struct {
    Game *game.Game
    Map  *gamemap.GameMap

    Units     map[game.UnitID]game.Unit
    Positions map[gamemap.ProvinceID]game.UnitID
    Orders    map[game.UnitID]game.Order

    MoveOrders        map[game.UnitID]game.MoveOrder
    HoldOrders        map[game.UnitID]game.HoldOrder
    SupportHoldOrders map[game.UnitID]game.SupportHoldOrder
    SupportMoveOrders map[game.UnitID]game.SupportMoveOrder
    ConvoyOrders      map[game.UnitID]game.ConvoyOrder
}
```

This avoids repeated type switches throughout the resolver.

---

## Multi-Pass Resolution Strategy

### 1. Normalize Orders

Every active unit receives an effective order.

- Submitted order exists: use it
- No submitted order: implicit Hold

```go
EffectiveOrders map[game.UnitID]game.Order
```

Default holds matter because other units may support them.

---

### 2. Categorize Orders

Split effective orders into typed maps:

```go
MoveOrders        map[game.UnitID]game.MoveOrder
HoldOrders        map[game.UnitID]game.HoldOrder
SupportHoldOrders map[game.UnitID]game.SupportHoldOrder
SupportMoveOrders map[game.UnitID]game.SupportMoveOrder
ConvoyOrders      map[game.UnitID]game.ConvoyOrder
```

---

### 3. Build Support Intents

Support orders describe intent, but they only apply if the supported unit's actual order matches that intent.

```go
type SupportIntent struct {
    Supporter     game.UnitID
    SupportedUnit game.UnitID
    Target        gamemap.ProvinceID
    Kind          SupportKind
    Applies       bool
    Cut           bool
}
```

Support-hold applies when the supported unit holds or otherwise remains in its province.

Support-move applies only when the supported unit has a matching move order:

```text
Support: A Gascony supports A Paris -> Burgundy
Actual:  A Paris -> Burgundy
Result: support applies

Support: A Gascony supports A Paris -> Burgundy
Actual:  A Paris -> Picardy
Result: support does not apply
```

---

### 4. Build Attacks

Every move order creates a potential attack.

```go
type Attack struct {
    Attacker game.UnitID
    From     gamemap.ProvinceID
    To       gamemap.ProvinceID
    ViaConvoy bool
}
```

Convoyed moves can also create attacks if a valid convoy route exists. Full convoy handling can be added after basic movement/support resolution.

---

### 5. Determine Support Cuts

Support is cut when the supporting unit is attacked by a foreign unit, except for rule-specific edge cases.

Common case:

```text
A Gascony supports A Paris -> Burgundy
A Marseilles -> Gascony
Result: support from Gascony is cut
```

Important details:

- Attacks by the same nation do not cut support
- Support is not cut by an attack from the province into which the support is being given
- Whether an attack succeeds is usually not required to cut support; the attack only needs to be a valid attack
- Convoy disruption and dislodgement can affect support in more advanced cases

Support cutting is one of the trickier parts of adjudication and should be built incrementally with tests.

---

### 6. Compute Strengths

All units have base strength 1.

Move strength:

```go
strength = 1 + number of uncut matching support-move orders
```

Hold/defense strength:

```go
strength = 1 + number of uncut matching support-hold orders
```

A unit that ordered a move generally does not defend its origin if it successfully moves away. If its move fails, its final position may still matter for occupancy and dislodgement.

---

### 7. Resolve Move Contests

Group attacks by target province.

```go
attacksByTarget map[gamemap.ProvinceID][]Attack
```

Basic rules:

- If one unit attacks an empty province and no equal-strength contest exists, it moves
- If multiple units attack the same province with equal highest strength, they bounce
- An attack against an occupied province must beat the defender's strength
- A unit cannot dislodge its own nation's unit
- Direct swaps fail unless performed by convoy
- Circular movement may succeed

The first implementation should focus on common cases, then add edge cases.

Suggested initial move-resolution scope:

1. Single move into empty province
2. Multiple equal-strength moves bounce
3. Attack against occupied province
4. Support strength
5. Dislodgement

Add later:

6. Direct swap detection
7. Circular movement
8. Convoy routes
9. Convoy disruption
10. Convoy paradoxes

---

## Convoy Resolution

Convoys require matching army move and fleet convoy orders.

Army move:

```go
MoveOrder{ViaConvoy: true}
```

Fleet convoy:

```go
ConvoyOrder{
    ConvoyedUnit: armyID,
    From: originProvince,
    To: destinationProvince,
}
```

A valid convoy route requires:

- the moving unit is an army
- the army starts in a coastal province
- the destination is a coastal province
- one or more fleets in water provinces issue matching convoy orders
- those fleets form a connected water route from origin to destination

Convoyed moves should be added after non-convoy movement and support adjudication works.

---

## Missing Orders

Units without submitted orders receive implicit holds during adjudication.

This should not mutate `Game.Orders`. It is an adjudicator normalization step.

---

## Applying a Resolution

After the adjudicator creates a `Resolution`, `Game.ApplyResolution(resolution)` should:

1. Move successful units
2. Update fleet coast positions
3. Mark or remove dislodged units from normal board occupancy
4. Create retreat requirements
5. Store the resolution for player display
6. Clear resolved unit orders
7. Advance the turn to retreat phase if retreats are required, otherwise to the next orders/adjustment phase

This should be implemented after `Resolve` can produce stable results.

---

## Testing Strategy

Build adjudication test-first.

Start with small scenario tests using the western Europe subset map. Then gradually add DATC-inspired cases.

Recommended initial tests:

1. Hold order succeeds
2. Single move into empty province succeeds
3. Two equal moves into same province bounce
4. Stronger supported move succeeds
5. Unsupported attack against occupied province fails
6. Supported attack dislodges defender
7. Support-hold increases defense strength
8. Support-move only applies when supported order matches
9. Support is cut by attack
10. Own-nation attack does not cut support

Convoy tests should come after basic movement/support tests.

---

## Implementation Order

1. Create adjudicator package
2. Define `Resolution`, `OrderResult`, `MoveResult`, `DislodgedUnit`, `RetreatRequired`
3. Build `resolutionContext`
4. Normalize missing orders to holds
5. Resolve simple moves without support
6. Add bounce/standoff detection
7. Add support strength
8. Add support cuts
9. Add dislodgement
10. Add retreat requirement output
11. Add `Game.ApplyResolution`
12. Add direct swaps and circular movement
13. Add convoy route resolution
