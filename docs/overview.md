# Diplomacy Architecture

## Overview
The purpose of this document is a brainstorm and plan the primary technical implementations of Diplomacy written in Go. This is intended to be a living document and will evolve as new ideas are explored.

## Resources
- [Diplomacy Rules](https://media.wizards.com/2015/downloads/ah/diplomacy_rules.pdf)
- [Adjudication Test Cases](https://webdiplomacy.net/doc/DATC_v3_0.html)

## Rules
- There are 7 great powers (England, Germany, Russia, Turkey, Italy, Austria-Hungary, France)
- Players are randomly assigned
- When a player controls 18 supply centers after the Fall turn or remaining players agree on a truce the game is over (latter in a draw)
- Three province types: inland, water, coastal. Coastal provinces may connect to water that cannot be connected together, ex. a northern and southern shore. Armies and fleets can occupy coasts
- 34 territories have supply centers. Players have armies and fleets equal to the number of supply centers as of the last Fall turn
- All units have the same strength
- Only one unit can be in a province at a time
- Units have 3 starting supply centers (Russia 4) and start with either the specified army or fleet on each of them
- Spring and Fall seasons. Each have 5 phases
  - Diplomacy
  - Write orders
  - Resolve orders
  - Retreat and disband
- Fall has extra of gaining and losing units
- Each unit gets max one order.

## Orders
- Move, Hold, Convoy, Support
- Only fleets can convoy

### Hold
- Keeps a unit in its place
- Not giving a unit an order defaults to Hold

### Move
- Move into a territory that is either free or occupied, in which case it's an attack
- Armies can move into Inland or Coastal provinces, not water
- Fleets can move into Water or Coastal provinces, not inland
- Fleets can only move from one Coastl province to another if they share a coastline
- If two units try to move into the same province with equal strength they simply do not move. This only applies if both are moving, not if one is already holding.
- If there is a chain of movement and the front of the chain stops there's a backup and none can move
- Units cannot trade places directly. They can move in a triangle (or larger shape) and rotate around.

### Support
- All units are equal strength so they require help (support) from other units. If an attack is successful the attacking unit moves into the territory. Defeated units without other move orders must retreat (later stage)
- Support can be offensive (supporting and attacking move) or defensive (support a hold/convoy)
- Support can be given without consent and cannot be refused
- Supporting units can only support provinces that they could have legally entered in that turn
- If intended supported unit is not moving then the supporter has to support the province and is only valid if the supported unit stays there. If it moves, the support is invalid
- If intended supported unit is moving then support is only valid if it matches the movement of the unit. If the unit moves somewhere else it is invalid.
- Units supporting another do not have to be adjacent to each other; the support focuses on the target province that is reachable
- If a unit is dislodged during resolution it still attacks where it was intended to move to *a different* province. But, if the dislodged unit attacks the province that dislodged it then there is no effect
- Support is cut if it is attacked from any province other than the one where support is given. This cancels the support order
- Support cannot be cut by a unit on also owned by that nation

### Convoy
- Basic convoy requires a fleet to remain stationary. An army can then move from one costal adjacent province to another one. Both army and fleet require a matching order.
- Can also move through multiple adjacent water spaces with the fleet following
- Can only convoy one army per turn
- Fleets in coastal province do not convoy. The fleet must be in a water province
- Convoy does not support an army
- A dislodged convoy causes the convoy to fail and the army remains in place. If an attack does not dislodge then the convoy is unaffected
- Units can trade places via convoy (but not directly on land)

### Retreat
- Retreats are written down and resolved immediately, without diplomacy
- Must retreat to a province that it could orginarily move to. Cannot retreat to: an occupied province, the province from which the attacker came, or a province left vacant by a standoff on the previous turn
- If two or more units retreat to the same province they are disbanded

## Gaining and Losing Units
- After the Fall phase a player controls a supply center if they have a unit on that province. It is then under their possention until taken by another player
- Based on supply centers controlled the players then disband or gain units
- Orders are given and resolved. New units can be placed on any free supply center owned by the player
- On a coastal province a fleet or army must be noted
- You can't build if your supply centers are occupied
- New units are only placed in the Home supply centers, not in captured ones

### General Mechanics
- A standoff means the strength of opposing forces are equal. It generally means "nothing happens"
- You cannot dislodge your own unit. Ex. if you try to move two armies into a space of another army you own, which attacks another province. If the attacking army is in a standoff then the two armies moving in don't succeed.
- You can attack a province you already own and even have a unit in to prevent it from being captured, ex. if the unit in that province attacks a different one
- You can have two units attack a province you already own to create a standoff and protect it from an adjacent threat.
- If a player leaves then it's considered that the government fell. Units simply stay put. If units must be removed then those furthest from the home center are removed first. Fleets before armies if a tie. Then alphabetically by province if another tie.

## Mechanics Brainstorming
- There might need to be a stage in the game state to accrue players, or this could be something that is more of a pre-game/web functionality
- Some kind of node graph to handle territories and adjacencies
- The game state itself doesn't have to follow the exact phases verbatim and will likely have more, though the players will not see those from their side
- Support will need to specify the intent
- Determine cut support as one of the first resolution items
- Some orders need to rely on expectations of other players, such as saying "My ally, the French, will convoy my army to another province", even if that player picks something else

## Questions to Answer
- What does the Order struct look like?
- How do we resolve all orders?
- What does the Resolution struct look like?
- Does the game start agnostic of players, or is there an "awaiting players" state? Should the web UI be responsible for player determination?
- What is the flow of player interation on the web side? How do we store the game state? 

## Architecture Tasks
- Determine the map graph and how to represent movement. The graph should just be the map nodes and how they connect, not anything related to units, supply centers, or current game state.

## Implementation Ideas to Explore
- Go embeded JSON
