# General Todo and Thoughts

- Worth looking into moving orders into a subdirectory of /game
- Move repositories into a different infrastructure directory, even if they're memory based
- How should the list of available retreats be stored? Current direction is to store a list of 1. attack origins and 2. standoff provinces. Alternatively, when we make a the retreat order we store a list of available retreats. This could then lead to another outcome, "forced disbandment"
- Add README for how the adjudicator actually works
- General simplification of data. Not sure a Unit should store current province ID. But if it should, then maybe the Unit should have a lot more current info, that way we don't need to perform as many lookups. Game adjudication then adjusts unit details. Examine tradeoffs.
