# Application Architecture

## Purpose

This document describes the intended structure of the Diplomacy application, the responsibilities of each layer, and how commands interact with the game workflow.

The central design rule is that the game engine owns game rules, while the application layer owns use-case orchestration. HTTP, databases, queues, and notification providers remain outside both.

## Layers

### Domain

The domain is the game engine under `core/`:

- `core/game` owns the `Game` aggregate, units, orders, turns, phases, and state mutations.
- `core/gamemap` owns map data and map-rule queries.
- `core/adjudicator` resolves submitted orders without performing persistence or external side effects.

Domain packages may depend on other domain packages, but they must not know about HTTP, users outside their game assignments, databases, queues, or notifications.

The domain is responsible for enforcing rules such as:

- whether an order is legal in the current phase;
- whether a unit belongs to the order's nation;
- whether movement is legal on the selected map;
- how orders resolve;
- whether a set of unit transforms is complete and valid;
- how valid transforms change game state;
- which phase follows the current phase.

### Application

The application layer implements complete use cases. It loads state, authorizes the actor, invokes domain behavior, persists the result, and records follow-up work or events.

Examples include:

- submitting an order;
- closing an order-submission phase;
- processing an automatic resolution phase;
- submitting a retreat;
- processing retreats;
- submitting an adjustment;
- processing adjustments.

The application layer may access user data and infrastructure through interfaces. It should not contain Diplomacy resolution rules or HTTP-specific behavior.

### Web

The web layer adapts HTTP requests to application commands. Its responsibilities are limited to:

- authenticating the request;
- parsing path, query, and body values;
- performing basic request-shape validation;
- constructing an application command;
- invoking the workflow;
- translating application errors and results into HTTP responses.

A web handler must not load and mutate a game directly.

### Infrastructure

Infrastructure packages implement interfaces required by the application layer. Expected implementations include:

- a game repository;
- a game-map repository;
- user storage;
- a durable job queue;
- a transactional outbox;
- notification delivery;
- a clock;
- database transaction management.

### Composition Root

`cmd/server/main.go` constructs concrete infrastructure implementations, injects them into the application workflow, and starts HTTP and worker processes. It is the only location that needs to know about all layers.

## Proposed Project Structure

```text
diplomacy/
├── cmd/
│   └── server/
│       └── main.go
├── core/
│   ├── adjudicator/
│   ├── game/
│   └── gamemap/
├── application/
│   └── gameplay/
│       ├── workflow.go
│       ├── repositories.go
│       ├── submit_order.go
│       ├── advance_game.go
│       └── events.go
├── web/
│   ├── handlers/
│   ├── middleware/
│   └── routes.go
├── infrastructure/
│   ├── postgres/
│   ├── jobs/
│   └── notifications/
└── docs/
```

The exact package names may evolve. The important constraint is dependency direction:

```text
web ───────────────┐
worker ────────────┼──> application ──> core
                   │         ↑
infrastructure ────┘         │
      implements application-owned interfaces
```

`core` imports neither `application` nor any outer package. `application` may import all domain packages. Infrastructure implements interfaces declared by the application package.

## GameWorkflow

`GameWorkflow` is an application service represented by a struct in the `application/gameplay` package. The package groups game-related use cases; the struct holds the dependencies shared by their handlers.

```go
type GameWorkflow struct {
    games      GameRepository
    maps       GameMapRepository
    users      UserRepository
    jobs       JobQueue
    events     EventPublisher
    transaction TransactionManager
}
```

The initial implementation does not need every dependency shown above. Dependencies should be added when a use case requires them rather than in anticipation of possible features.

`GameWorkflow` is not a global singleton and does not contain the currently active game. It is safe for one workflow value to process many games. The singular identity of a workflow operation is the game ID plus the state version loaded from persistence.

There is no need for a separate `GameManager` at first. The workflow already fills that role at the application level. A manager abstraction should only be introduced later if it acquires a distinct responsibility.

### Lifetime and Concurrency

Create one long-lived `GameWorkflow` when the process starts and inject its pointer into HTTP handlers and workers:

```go
workflow := gameplay.NewGameWorkflow(games, maps, jobs, events)
handler := web.NewGameHandler(workflow)
```

The workflow remains in memory for the life of the process, but it must not retain mutable state for individual games or requests. Its fields are shared dependencies such as repositories and queue clients. Those implementations must be safe for concurrent use.

Go's HTTP server already handles concurrent requests in separate goroutines. Handlers should call workflow methods directly rather than creating another goroutine for each command. Every method should accept `context.Context` so cancellation and deadlines propagate to repositories and other dependencies.

Concurrent operations on the same game are coordinated through persisted versions and transactions, not through mutexes held by the workflow. An in-memory repository used in development or tests may need a mutex to provide the same concurrency guarantees.

Goroutines are appropriate for long-running worker loops and bounded concurrent job processing. Command methods should not start fire-and-forget goroutines for notifications, phase advancement, or other durable work. They should record that work in the transactional outbox so it survives process crashes and can be retried.

## Commands

A command is a plain data structure describing an application request. It should not contain repositories, perform work itself, or require an `Execute` interface.

```go
type SubmitOrderCommand struct {
    GameID          game.GameID
    PlayerID        game.PlayerID
    ExpectedVersion uint64
    Order           game.Order
}
```

The command includes identity and concurrency information needed to execute the use case. The authenticated web handler supplies `PlayerID`; it must not trust a separate player ID supplied by the request body.

Commands should not receive `*game.Game` from the web or worker layer. If callers load the game themselves, authorization, persistence, concurrency, and transaction handling become spread across multiple layers. Instead, a workflow method receives a command and loads the aggregate itself:

```go
func (w *GameWorkflow) SubmitOrder(ctx context.Context, cmd SubmitOrderCommand) error
```

Inside a workflow method, private helpers may accept a loaded `*game.Game` when that makes the code clearer.

Separate command-handler structs are unnecessary while handlers remain small. If a workflow grows too large, each command may later become its own handler struct without changing command inputs or domain APIs.

## Validation Responsibilities

Validation occurs at three boundaries:

1. **Web validation** checks that the request can be parsed and required fields are present.
2. **Application validation** checks actor authorization, game membership, expected version, and whether the requested use case is permitted.
3. **Domain validation** checks Diplomacy rules and game-state invariants.

For example, the application verifies that the authenticated player is assigned to the nation issuing an order. `Game.SubmitOrder` verifies that the unit belongs to that nation and that the order is legal on the game map.

Duplicating a cheap security-sensitive check at adjacent boundaries is acceptable, but game rules should have one authoritative implementation in the domain.

## Submit Order Flow

A `SubmitOrder` request should follow this sequence:

1. The web layer authenticates the user and parses the order.
2. The web layer constructs `SubmitOrderCommand` using the authenticated player ID.
3. `GameWorkflow.SubmitOrder` begins a transaction or unit of work.
4. The workflow loads the game and its current version.
5. The workflow checks that the expected version matches.
6. The workflow authorizes the player against the game's nation assignments.
7. The workflow loads the game's map.
8. The workflow calls `game.SubmitOrder(order, gameMap)`.
9. The workflow saves the game using optimistic concurrency.
10. If submission completes the phase, the workflow records an advancement job in the transactional outbox.
11. The transaction commits.

The repository save should resemble:

```go
Save(ctx context.Context, g *game.Game, expectedVersion uint64) (newVersion uint64, err error)
```

If another operation already changed the game, the save returns a concurrency error rather than overwriting newer state.

Whether all players must explicitly lock their orders or submission alone can complete a phase is a separate domain/application policy. Both policies can enqueue the same advancement command.

## Advancing and Processing Phases

Phase advancement is different from player commands because it may trigger automatic business logic. It should still enter through the workflow rather than through a web handler mutating `Game` directly.

A durable job carries a command such as:

```go
type AdvanceGameCommand struct {
    GameID          game.GameID
    ExpectedVersion uint64
    Trigger         AdvanceTrigger
}
```

Possible triggers include all players being ready, a deadline expiring, or an administrative action. The trigger is useful for authorization and auditing; it must not alter the game rules applied.

Each job should process one durable state transition. The workflow loads the game, verifies its version, and switches on the current phase:

### AcceptOrders

- Verify that the trigger is allowed: all required players are ready, the deadline expired, or an administrator forced advancement.
- Advance to `ResolveOrders`.
- Persist the game.
- Enqueue processing for the new version.

### ResolveOrders

- Load the game map.
- Call `adjudicator.Resolve(game, gameMap)`.
- Pass every returned `game.UnitTransform` to `game.ApplyUnitTransforms`.
- Persist the complete adjudication result for history and player display.
- Advance to `AcceptRetreats` when retreats exist, or to the appropriate following phase when none exist.
- Persist the game and publish an `OrdersResolved` event.
- Schedule any required deadline or immediate follow-up job.

### AcceptRetreats and AcceptAdjustments

- Wait for player input, readiness, or a deadline.
- Accept commands through dedicated workflow methods.
- Once complete, advance to the matching resolution phase and enqueue it.

### ResolveRetreats and ResolveAdjustments

- Resolve the complete submitted set.
- Apply the result atomically to the game.
- Advance to the next input phase.
- Persist events and any next deadline.

Automatic phases may eventually be processed in the same transaction as the transition into them, but separate durable jobs are initially easier to retry, observe, and recover. A game temporarily being in `ResolveOrders` is safe as long as the outbox guarantees that its processing job will be delivered.

## Versions and Concurrency

A game version is a monotonically increasing persistence value. It does not need to be part of the `Game` domain struct unless domain behavior depends on it.

Every modifying command carries the version observed by its caller or scheduler. A repository update succeeds only when the stored version matches:

```sql
UPDATE games
SET state = ?, version = version + 1
WHERE id = ? AND version = ?
```

If no row is updated, the operation is stale. This prevents two web requests, duplicate jobs, or a deadline job and readiness job from resolving the same phase twice.

Queued phase jobs should contain the version created when they were scheduled. If a worker receives a stale job, it can safely acknowledge it without changing the game.

## Transactions, Jobs, and Events

Saving a game and scheduling follow-up work must be atomic. Otherwise the game could be saved in `ResolveOrders` while the process crashes before enqueueing its resolver job.

Use a transactional outbox:

1. Save the updated game.
2. Insert jobs and events into outbox rows in the same database transaction.
3. Commit.
4. An infrastructure worker publishes pending outbox rows to the job queue or notification system.

Application code emits facts such as:

- `OrdersRequested`;
- `OrdersResolved`;
- `RetreatsRequested`;
- `PhaseAdvanced`;
- `GameCompleted`.

Notification handlers consume those events. `GameWorkflow` should not send email or websocket messages directly because external delivery cannot participate reliably in the game-state transaction.

The full adjudicator resolution, including order outcomes and reason codes, should be stored as history or as event data. Only unit transforms need to be applied to the `Game` aggregate.

## Repository Interfaces

Interfaces belong to the application package that consumes them, not to infrastructure packages.

A minimal starting point is:

```go
type StoredGame struct {
    Game    *game.Game
    Version uint64
}

type GameRepository interface {
    Get(ctx context.Context, id game.GameID) (StoredGame, error)
    Save(ctx context.Context, g *game.Game, expectedVersion uint64) (uint64, error)
}

type GameMapRepository interface {
    Get(ctx context.Context, id gamemap.MapID) (*gamemap.GameMap, error)
}
```

Tests can provide in-memory implementations. Postgres and embedded-map implementations belong in infrastructure packages.

Transaction handling may later require repository methods to operate through a unit-of-work value. That choice should follow the selected database library rather than being abstracted prematurely.

## Error Handling

Application errors should distinguish conditions the outer layer needs to map differently:

- unauthenticated or unauthorized actor;
- game or map not found;
- stale version;
- invalid command;
- domain-rule violation;
- transient infrastructure failure.

The web layer maps these categories to HTTP responses. Workers use them to decide whether a job should be acknowledged, retried, or moved to a dead-letter queue.

Domain errors should remain meaningful without containing HTTP status codes.

## Testing Strategy

### Domain tests

Test game rules directly with in-memory values. These tests cover order validation, adjudication, transforms, retreats, adjustments, and phase transitions without repositories or HTTP.

### Application tests

Construct `GameWorkflow` with fake repositories, maps, queues, and event publishers. Verify complete use cases:

- authorization occurs before mutation;
- the correct domain method is invoked through observable state changes;
- successful commands save with the expected version;
- stale commands do not mutate persisted state;
- adjudication results are applied and recorded;
- follow-up jobs and events are emitted;
- failures do not partially save state or enqueue work.

### Web tests

Test request parsing, authentication integration, command construction, and error-to-response mapping. Do not repeat domain adjudication cases through HTTP.

### Infrastructure tests

Use integration tests for optimistic updates, transactions, outbox delivery, serialization, and repository round trips.

## Initial Implementation Sequence

1. Create `application/gameplay` and define `GameWorkflow`, `GameRepository`, and `GameMapRepository`.
2. Implement `SubmitOrderCommand` and its workflow method using in-memory test doubles.
3. Add versioned repository behavior.
4. Implement `AdvanceGameCommand` for `AcceptOrders` and `ResolveOrders`.
5. Persist adjudication history and enqueue follow-up work through an outbox abstraction.
6. Add thin HTTP handlers and a worker adapter.
7. Extend the same workflow for retreats and adjustments.

This sequence creates one complete order-submission and resolution path before adding more domain phases or production infrastructure.

## Decisions to Keep Explicit

The following policies should remain explicit application or domain decisions rather than emerging accidentally from infrastructure behavior:

- whether order completion requires explicit player readiness;
- whether no-retreat turns skip the retreat input phase;
- deadline ownership and duration;
- administrative force-advance permissions;
- retry and dead-letter behavior;
- how adjudication history is presented and retained;
- whether automatic phases use separate jobs or execute immediately.
