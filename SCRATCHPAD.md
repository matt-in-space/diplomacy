# Update Sidebar Data
- Whenever the newly added web/static/js/gameClient.mjs receives game data we need to not only draw the map but also update the current game state display
- This includes web/templates/game.html where the current turn, list of players, and the player's own units are displayed
- The game.onUpdate() callback needs to access the game state to update the sidebar
- Get the container elements .turn-status, .nation-list, and .unit-list
- Every time the game state is updated, use the game state to update the sidebar
