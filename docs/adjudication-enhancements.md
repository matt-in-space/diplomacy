# Adjudication Enhancements

Deferred work and possible improvements for the adjudicator, kept separate so the
core resolver (see `adjudication-resolver.md`) can stay focused.

## Deferred features

- **Retreat-destination computation.** v1 marks dislodged units with a `retreat`
  outcome and `To=""`. Computing the set of legal retreat provinces (excluding the
  attacker's origin, occupied provinces, and standoff provinces) belongs to the
  retreat phase.
- **Dislodged-by bookkeeping in output.** The resolver knows which unit dislodged
  which; surfacing that in the `Resolution` (e.g. for UI or retreat constraints)
  is not yet exposed.
- **Build / disband phase.** Adjustment-phase adjudication (supply-center counts,
  builds, disbands) is out of scope for the movement resolver.
- **Richer reason taxonomy.** More granular `ReasonCode`s (e.g. distinguishing a
  bounce from a beleaguered garrison, or the specific convoy-paradox rule applied).

## Possible improvements

- **Alternative backup rules.** v1 uses the Szykman rule for convoy paradoxes.
  Other rule sets (e.g. "all-hold", the 1971/2000 rulebook variations) could be
  made configurable.
- **Bicoastal coast edge cases.** Confirm coast resolution on fleet moves handles
  every bicoastal scenario; add tests if gaps appear.
- **Performance.** The resolver is recursive with memoization; if turn sizes grow,
  profile the guess/backtrack paths and the convoy path search.
- **Dependency visualization.** The (now removed) dependency graph could return as
  an optional diagnostic that renders per-turn order dependencies, independent of
  the resolution engine.
