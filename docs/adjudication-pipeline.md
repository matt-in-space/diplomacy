# Adjudication Pipline

## Overview
The key component and challenge of a Diplomacy game engine is the adjudication pipline. While the rules are relatively simple, all unit orders must be resolved at once rather than sequentially (at least as far as the user experiences it). There are also numerous edge cases to consider. Still, by following a simple, logic path I believe most of the edge cases will be handled. Additionally, it should be built in a way that, if a new edge case is discovered, it can be easily added without modifying the existing logic.

## Implmentation
### Setup
#### Game data structure
The primary entity is the `Game` struct. It is the container for all game state and includes methods that allow users to submit unit orders. Logically, there is a difference between the act of submitting orders versus the act of resolving them. When an order is submitted via the `game.SubmitOrder()` method, it is validated that the order can actually be submitted based on the allowed submission parameters, ex. that the unit exists and is owned by the submitting player nation. Adjudication is the act of actually resolving them. For example, you can legally submit an order for two units to trade places. But, it is the adjudication process that determines that they cannot actually do that.

In addition to the `Game` there is a `GameMap`game. This is primarily a reference object that denotes all of the provices and connections between them. It is referenced by the `Game`, but is not part of if. When adding orders or adjudicating the game the `GameMap` is passed in to the appropriate functions.

The `Game` also stores the current state of units on the map. The primary record is the `Unit`, which is a struct noting the unit type, location, and owner. The `Game` includes a few lookup tables to efficiently access necessary data. These are:

- `Units`: A map of units by their ID
- `Positions`: A reverse lookup table of units by their location
- `FleetCoasts`: Fleet units are additionally positioned on a specific coast when in a water or coastal province. This acts as a secondary graph for fleet movements.

Players then submit individual orders via the `Game` object and they are stored on the `Orders` field, which is a map of orders by unit ID.

The general idea of Diplomacy is simple. Players discuss and then submit orders. Orders are resolved and the above game state will be modified accordingly.

It should be assumed that the state of the game is always valid. As the primary object anything that modifies the game state will be checked for validity. Whenever the game state is passed into external functions we can make the assumption that we don't need to perform any additional validation.

#### Adjudication process
The adjudicator is a separate package outside of the game. Its only role is to receive the game state and the players orders then return "what happened" as a separate struct. It does not modify the game state itself.## Interface

### Interface
The proposed interface is defined in the `adjudicator` package.

```go
func Resolve(g *Game, gm *GameMap) (Resolution, error) {
	// ...
}

type Resolution struct {
	Turn game.Turn
	Outcomes map[game.UnitID]Outcome
}

type Outcome struct {
	UnitID game.UnitID
	Unit UnitOutcome
	Order OrderOutcome
}

type UnitOutcomeType string

const (
	UnitOutcomeMove    UnitOutcomeType = "move"
	UnitOutcomeHold    UnitOutcomeType = "hold"
	UnitOutcomeRetreat UnitOutcomeType = "retreat"
)

type UnitOutcome struct {
	UnitID game.UnitID
	Type 	 UnitOutcomeType
	From   gamemap.ProvinceID
	To     gamemap.ProvinceID
	Coast  gamemap.CoastID
}

type OrderOutcome struct {
	Order   game.Order
	Success bool
	Reason  ReasonCode
}
```

Every unit on the game board will have a single corresponding order (defaulting to a hold order). The output Resolution maps every unit ID to the corresponding outcome, which is split by the unit outcome and the order outcome.

The unit outcome denotes the final state of the unit after resolution. If it attempted to move but was unable to because of a standoff the outcome type is "hold". If the unit was displaced, it's ending province is currently the same as where it started but the outcome type is to "retreat", which will require futher orders in the next phase of the turn.

The order outcome is related, but denotes whether or the the user's order was fulfilled as intended. If a user intended to support an ally but that ally did not move their unit to the expected location the unit outcome is the same, but the order itself failed.

Reason codes for order outcome failures will be determined as needed during implementation.

### Resolution Logic

Resolution follows simple path to produce the output. The general idea is that we build up a lookup table for every unit's intended ending position. Assuming all convoy orders were valid and no province ended with multiple units, we can simply map each unit to its intended ending position. Basic conflict resolution will cover the vast majority of cases. We will need to handle all the non-conflicting cases first, then resolve conflicts.

#### Normalize Orders
Every unit needs an order. If a unit does not have an order, it is treated as a hold. Create a list of effective orders for each unit.

#### Build resolution state map
Every unit needs to simply track whether or not the order has been resolved or not. A simple map of unit ID to boolean value will suffice.

#### Build map of intended ending positions
Create a map that associates each unit with its intended ending position. Move orders end with the target location. Hold, convoy, and support orders end with the unit's current location.

#### Build a map of effective support
Any supporting units will only provide support if the unit they are supporting matches the expected move (either a move or hold order), and the support is not cut off by an attack. Support orders that are invalid will be resolved as failed with relevant reason codes.

#### Build map of effective convoys
Similar to support, any convoy orders will only be valid if the unit they are convoying matches the expected move and the convoy is not cut off by an attack. If a convoy fails then all fleets involved in the convoy will be treated as failed.

#### Resolve all indended movements to provinces with a single final unit
We can assume any movements that result in a single unit in a province are valid if the unit can move to that province. These units can all be resolved to their final positions.

#### Calculate strengths and resolve conflicts
All remaining units should be move attempts with or without support. For each move, calculate the strength of the unit and resolve any conflicts with other units in the same province. Units can either move, hold, or be dislodged and will require a retreat order in the subsequent game phase.

### Implementation Order
The internal resolution model should be designed up front, but implemented incrementally. `Resolve` must remain non-mutating throughout, and each phase should have independent tests before proceeding to the next.

#### Phase 1: Foundations and basic movement
- Define the resolution, outcome, and reason-code types.
- Validate resolver inputs, including the game map and `ResolveOrders` phase.
- Normalize missing orders to implicit holds without modifying the game.
- Categorize effective orders into lookup-friendly internal structures.
- Resolve uncontested moves, competing moves, and attacks against occupied provinces.
- Add dependency handling for move chains, direct swaps, and circular movement.

#### Phase 2: Supports and dislodgement
- Match support intents against effective orders.
- Treat support as support into a province rather than requiring coast notation to match the supported move.
- Determine support cuts, including the foreign-attack and supported-province exceptions.
- Calculate attack and defense strength.
- Enforce the rule that a nation cannot dislodge its own unit.
- Produce dislodgement and retreat outcomes.
- Align support-order submission validation with province-based support matching.

#### Phase 3: Ordinary convoys
- Match convoyed army moves with fleet convoy orders.
- Find complete and alternate routes through convoying fleets in water provinces.
- Permit convoyed direct swaps.
- Disrupt a route only when a fleet on that route is dislodged.
- Allow a convoy to succeed when at least one complete route remains intact.

#### Phase 4: Convoy paradox detection
- Re-evaluate convoy routes, fleet dislodgements, support cuts, and movement outcomes until the result stabilizes.
- Detect repeated dependency states instead of looping indefinitely.
- Return a defined error for convoy paradoxes.
- Defer choosing and implementing a paradox-resolution policy until ordinary adjudication is stable.

## Tests
### Test Setup
The test setup can load a game with the includeed Western Europe data file. This will start with two nations and three game units. For each test we will need to manually adjust the game state as described above, if needed, and add the necessary orders. We will then assert that the result of the adjudication pipeline matches the expected outcome.

### Test Cases
The following scenarios should be tested for:
- Units without orders default to hold
- A single unit moves into an unoccupied province
- Two units on the same nation cannot directly trade positions
- Units can move into a circle to trade positions (ex a triangle)
- A unit attacking a province with one defender results in a draw
- A supported unit attacking a single defender results in the defender retret
- A supported unit attacking another supported unit results in a draw
- A supported unit receives no support if the supporting unit is cut. Results in a draw
- A support order fails if the unit does not hold in the expected province
- A support order fails if the supported unit does not move to the expected province
- A unit can move across a full convoy
- A unit cannot move if any of the convoy units are cut
- A unit cannot move if any of the convoy units do not respect the convoy
- A unit can move across multiple water spaces if supported by convoys
