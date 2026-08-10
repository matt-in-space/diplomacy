// The player-scoped slice of an active game's state — see
// application/gameplay/player_view.go's PlayerView for the authoritative
// shape (Turn, Units, SupplyCenterOwners, CommittedNations, YourNation,
// YourOrders). Fetched fresh per page load from GET /games/{id}/state,
// never cached like the static map geometry is — this changes as the game
// progresses.
import { createStore } from "./store.mjs";

export async function loadGameState(gameId) {
	const res = await fetch(`/games/${gameId}/state`);
	if (!res.ok) {
		throw new Error(`failed to load game state ${gameId}: ${res.status}`);
	}
	return await res.json();
}

// The single instance the rest of the frontend reads/subscribes to. Starts
// null until main() loads and sets the real value — nothing consumes this
// yet (no listeners wired up), it's just where the fetched state lives.
export const gameState = createStore(null);
