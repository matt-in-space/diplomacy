# Order Flow

We have 3 phases where orders are received and resolved.

1. Unit orders
2. Retreat orders
3. Adjustment orders

Each order phase has two concerns:

- **Input phase** — players/nations submit orders
- **Resolution phase** — the engine resolves the submitted orders and mutates the game state

The core engine should remain frontend-agnostic. UI concerns such as whether a player has viewed results, unread notifications, or whether a modal should be displayed belong in the web/application layer.

## Unit Orders

An order will be applied to the `Game` struct using `SubmitOrder()`. Only a single order is submitted at a time to simplify validation. There is a different struct for each order type.

Orders belong to **nations**, not players. A player may submit an HTTP request, but the engine rules are about nations and units.

```go
type NationID string
type UnitID string
```

The command/application layer is responsible for verifying that a player controls the nation they are submitting for:

```text
SubmitOrderCommand{GameID, PlayerID, Order}
  → load game
  → verify PlayerID controls Order.Nation
  → load map using game.MapID
  → game.SubmitOrder(order, gameMap)
  → save game
```

`Game.SubmitOrder()` should not know about `PlayerID`. It validates game rules only.

## Order Validation

The order must be validated against the game state to be accepted. In general the UI should prevent invalid orders from being submitted, but engine validation is still authoritative.

Validation includes:

1. The game is currently accepting unit orders
2. The order's nation exists in the game
3. The ordered unit exists
4. The ordered unit belongs to the order's nation
5. Only one active order exists per unit
6. Movement must be valid using the game's map
7. All referenced units, provinces, and coasts must exist

Submitting an order for a unit that already has an order should replace the existing order while the phase is still open. Once orders are locked or the phase advances, replacement should be rejected.

At resolution time, any unit without an explicit order defaults to Hold.

## Game State Needed for Orders

The `Game` struct needs enough state to validate and store orders, but it should not own static map data.

Suggested core state:

```go
type Game struct {
    ID            GameID
    MapID         gamemap.MapID
    Turn          Turn
    Units         map[UnitID]Unit
    SupplyCenters map[gamemap.ProvinceID]NationID
    Orders        map[UnitID]Order
}
```

Units belong to nations and occupy provinces. Fleets may also need a coast value when their location is coast-specific.

```go
type UnitType string

const (
    Army  UnitType = "army"
    Fleet UnitType = "fleet"
)

type Unit struct {
    ID       UnitID
    Nation   NationID
    Type     UnitType
    Province gamemap.ProvinceID
    Coast    gamemap.CoastID
}
```

Supply center ownership is mutable game state and should be stored separately from map data. Map data only defines which provinces are supply centers and which nations, if any, treat them as home centers.

## Map Reference

A game should reference its map by ID, not store the full map graph inside the game state.

```go
type MapID string
```

The application layer or a future `GameMapManager` can load/cache maps by ID and pass the resolved `*gamemap.GameMap` into game methods that need map rules.

```go
game.SubmitOrder(order, gameMap)
game.Advance(gameMap)
```

This keeps the game state serializable and prevents duplicating static map data across every saved game.

## Player and Nation Assignment

Authorization should live above the game engine. The web/command layer verifies that a `PlayerID` may submit orders for a `NationID`.

The game engine may eventually store minimal assignment metadata for persistence or self-contained game state, but `Game.SubmitOrder()` should not require `PlayerID`.

Possible application-level model:

```go
type PlayerID string

type Assignment struct {
    Nation NationID
    Player PlayerID
}
```

The engine itself should continue to model rule ownership in terms of nations:

- Units belong to nations
- Supply centers belong to nations
- Orders belong to nations

## Should Orders Target Units or Provinces?

The ordered unit should always be identified by `UnitID` because units move and province ownership changes.

Order targets depend on the order type:

- Hold targets only the ordered unit
- Move targets a province and, for fleets entering bicoastal provinces, possibly a destination coast
- Support targets another unit and the province/coast being supported
- Convoy targets an army and its intended movement

Example shape:

```go
type MoveOrder struct {
    Nation NationID
    Unit   UnitID
    To     gamemap.ProvinceID
    Coast  gamemap.CoastID
}
```

Support and convoy orders likely need both unit references and movement intent because they are only valid when they match the intended order.

## Questions Answered

### Should we allow multiple orders per unit and simply replace the existing order?

Yes. While the game is accepting orders, submitting a new order for the same unit replaces the previous one. The storage shape should enforce one active order per unit:

```go
Orders map[UnitID]Order
```

### How should players be linked to the game state?

Player authorization should be handled by the command/application layer, not `Game.SubmitOrder()`.

The game may store assignments later if it is useful for persistence, but the core rules should only require nations.

### Should orders target Units or Provinces?

Both, depending on the order type. The ordered piece should always be a `UnitID`. Movement destinations should be provinces/coasts. Support and convoy orders should reference the affected unit plus the relevant movement intent.

## Implementation Order

1. Add core domain IDs and unit types:
   - `GameID`
   - `NationID`
   - `UnitID`
   - `UnitType`
   - `Unit`
2. Expand `Game` with:
   - `MapID`
   - `Turn`
   - `Units`
   - `SupplyCenters`
   - `Orders`
3. Define unit order structs:
   - `HoldOrder`
   - `MoveOrder`
   - `SupportOrder`
   - `ConvoyOrder`
4. Define the `Order` interface with methods needed for validation and storage.
5. Implement `Game.SubmitOrder(order, gameMap)`.
6. Add validation tests for:
   - wrong phase
   - unknown nation
   - unknown unit
   - unit owned by another nation
   - replacement of existing unit order
   - invalid army/fleet movement
   - missing province/coast references
7. Defer retreat and adjustment order types until unit orders and basic adjudication shape are stable.
