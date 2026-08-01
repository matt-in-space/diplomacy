# General Todo and Thoughts

- Worth looking into moving orders into a subdirectory of /game
- Move repositories into a different infrastructure directory, even if they're memory based
- Once the UI exists, add an application-layer history store (persisted per-phase Resolution, e.g. as JSON) so players can review past turns. Game itself intentionally doesn't hold this beyond LastOrderResolution/LastRetreatResolution, which exist for retreat-legality rules, not history.
- Add README for how the adjudicator actually works
- General simplification of data
