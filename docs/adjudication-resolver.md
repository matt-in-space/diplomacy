# Adjudication Resolver

This document describes the resolution engine that turns a set of validated,
pruned orders into a `Resolution`. It replaces the earlier topological
dependency-graph idea, which cannot correctly resolve the cyclic dependencies
that Diplomacy allows (convoy paradoxes, circular movement).

The engine is a recursive, backtracking resolver based on Lucas Kruijswijk's
"The Math of Adjudication".

## 0. Cleanup & prerequisites

- Remove the dependency-graph machinery (`buildDependencyGraph`, `addDependency`,
  `providesSupportOrConvoy`, the `dependents`/`indegree` fields, and their tests).
  `currentPositions` stays — the resolver uses it.
- Rebuild `intendedPositions` from the effective orders after pruning (moves →
  target province; holds/supports/convoys → current province). Drop the stale
  pre-prune build.
- Add resolver state to `resolutionContext` (see below).
- Track deferred work in `docs/adjudication-enhancements.md`.

### File layout

```
adjudicator.go  -> Resolve orchestration + preprocessing
resolver.go     -> state model, resolve(), adjudicate(), backupRule()
strength.go     -> path/hold/attack/defend/prevent, support-cut, dislodgement helpers
outcome.go      -> Resolution population
gamemap         -> ConvoyPathExists (done)
```

## 1. Node & state model

Every unit's effective order is a resolvable node identified by `UnitID`.

```go
type resolutionState int

const (
    stateUnresolved resolutionState = iota
    stateGuessing
    stateResolved
)

// on resolutionContext:
state           map[game.UnitID]resolutionState
resolution      map[game.UnitID]bool
dependencyStack []game.UnitID
```

`resolution[u]` meaning by effective order:

- move → the unit successfully moves to its destination
- support → the support is given (not cut, supporter not dislodged)
- convoy → the convoying fleet stands (not dislodged)
- hold → the unit holds (not dislodged)

Dislodgement is derived afterward from which moves succeeded.

## 2. resolve() recursion + backtracking

```
resolve(u):
    if state[u] == Resolved: return resolution[u]
    if state[u] == Guessing:            # reached a node mid-guess => cycle
        record u in dependencyStack
        return resolution[u]

    oldStackLen = len(dependencyStack)
    resolution[u] = false               # first hypothesis: FAILS
    state[u] = Guessing
    firstResult = adjudicate(u)

    if len(dependencyStack) == oldStackLen:   # no cycle touched this node
        state[u] = Resolved; resolution[u] = firstResult
        return firstResult

    if dependencyStack[oldStackLen] != u:     # in a cycle but not its entry point
        record u in dependencyStack
        resolution[u] = firstResult
        return firstResult

    # u is the entry point of a cycle: test the second hypothesis
    truncate dependencyStack to oldStackLen
    state[u] = Guessing
    resolution[u] = true                # second hypothesis: SUCCEEDS
    secondResult = adjudicate(u)

    if firstResult == secondResult:     # both hypotheses agree => stable
        truncate dependencyStack to oldStackLen
        state[u] = Resolved; resolution[u] = firstResult
        return firstResult

    # genuine paradox/cycle: apply backup rule, then retry
    backupRule(cycle members since oldStackLen)
    truncate dependencyStack to oldStackLen
    return resolve(u)
```

Top level: call `resolve(u)` for every unit; order does not matter (memoized).

## 3. Backup rule (cycle resolution)

`backupRule(cycle)` force-resolves the members of a genuine cycle:

- All members are move orders → circular movement: mark all resolved `true`.
- Cycle contains a convoy/support paradox → Szykman rule: force the paradoxical
  convoyed move(s) to resolved `false` (the convoy fails), so support-cut
  resolves consistently.

## 4. adjudicate(u) by order type

- move → move-success per the strength comparison (§5)
- support → not cut: `true` unless supporter dislodged or a qualifying foreign
  attacker cuts it (§6)
- convoy → fleet stands: `true` unless a move into the fleet's province succeeds
- hold → holds: `true` unless a move into its province succeeds

Each recurses via `resolve(...)` into the moves/supports it depends on.

## 5. Strength calculators

For move `m`: unit `u`, nation `N`, destination `D`, occupant `occ`.

- path(m): non-convoy → adjacency (guaranteed by validation). Convoy → a chain of
  non-dislodged convoying fleets exists from origin to `D` (`ConvoyPathExists`
  filtered to fleets whose resolve = stands).
- holdStrength(D): empty → 0; occupant moves away successfully → 0; occupant
  ordered to move but fails → 1; else → 1 + valid support-to-hold count.
- attackStrength(m): path fails → 0. If `D` empty or `occ` vacates
  (non-head-to-head) → 1 + valid supports. Else if `occ.nation == N` → 0 (no
  self-dislodgement); else → 1 + valid supports whose supporter's nation ≠
  `occ.nation`.
- defendStrength(m): 1 + valid supports (head-to-head only).
- preventStrength(m): path fails → 0; head-to-head and opponent succeeds → 0;
  else → 1 + valid supports.
- move `m` succeeds iff: path holds, attackStrength(m) > 0, attackStrength(m) >
  every competing move's preventStrength, and attackStrength(m) >
  (defendStrength(opponent) if head-to-head else holdStrength(D)). Ties fail.

"Valid support" = a support order for `m` whose resolve = given. Self-dislodge
protection, support-can't-help-dislodge-own, beleaguered garrison, standoffs, and
head-to-head all emerge from these definitions.

## 6. Support-cut predicate

Support `s` by unit `w` (province `q`), supporting an action possibly targeting
province `T`:

- Cut iff there exists a move by unit `a` with `a.nation != w.nation`, into `q`,
  with path(a) holding, and (if `s` supports a move into `T`) `a.origin != T`.
- Only requires the attacker to reach `q` (not to win); own-nation excluded; the
  "attack from the supported province" exception applied. Supporter dislodged
  voids it separately.

## 7. Convoy path with dislodgement

When adjudicating a convoyed move, assemble the path from convoying fleets whose
resolve = stands (not dislodged). No such chain to the destination → path fails,
army holds (`convoy_failure`). This recursion is the convoy-paradox entry point,
handled by the backup rule (§3).

## 8. Dislodgement derivation

After all resolves: a unit at `p` is dislodged iff some move into `p` resolved
true and the unit did not itself move out. Record the dislodger (future retreat
rules; v1 only marks `retreat`).

## 9. Outcome population

One `Outcome` per unit:

| Effective order | Result | UnitOutcome | OrderOutcome |
|---|---|---|---|
| move succeeds | moved | move, To=dest, coast resolved | success |
| move fails, not dislodged | bounced | hold at origin | fail, weak_attack |
| any order, dislodged | dislodged | retreat, From=prov, To="" | fail, dislodged |
| support given | — | hold | success |
| support cut | — | hold | fail, support_cut |
| convoy stands | — | hold | success |
| hold, not dislodged | — | hold | success |
| pruned (misaligned/convoy-fail) | — | hold | keep existing outcome |

- Fleet coast on a successful move: use `MoveOrder.TargetCoast`; fill single-coast
  provinces from the map.
- Merge, don't clobber outcomes already written during pruning.
- Retreat scope: mark `retreat` with `To=""`, no destination search (retreat
  phase, later).
- `Resolve` returns `Resolution{Turn: g.Turn, Outcomes: ...}`.

## 10. v1 scope vs. enhancements

In v1 (correct core): holds, moves, bounces, standoffs, supports + cut (with
exceptions), dislodgement, self-dislodge protection,
support-can't-help-dislodge-own, beleaguered garrison, head-to-head, convoy
movement, convoy disruption by dislodgement, circular movement, convoy paradox
(Szykman).

Deferred (see `docs/adjudication-enhancements.md`): retreat-destination
computation, disband/build phases, richer reason taxonomy, dislodged-by
bookkeeping in output, performance passes, alternative backup rules.

## 11. Testing plan

- Audit the black-box scenarios (LLM-written) against the rules before trusting
  them as the oracle.
- Bring-up order: defaults/holds → simple moves → standoffs/circular → strength &
  support → support cut → dislodgement/head-to-head → convoy movement → convoy
  disruption → paradoxes.
- Add internal DATC-style tests (`strength_test.go`, `resolver_test.go`) using the
  `testUnit` builders, plus targeted paradox cases (DATC 6.F convoy, 6.G circular).
