// The single place the frontend talks to the server about one specific
// game. Callers never need to remember to also sync gameState themselves
// — refresh() does both the fetch and the store update as one call.
import { loadGameState, gameState } from "./gameState.mjs";

export function createGameClient(gameId) {
	return {
		// Reloads game state from the server and updates the shared
		// gameState store. Errors propagate rather than being caught here
		// — same reasoning as loadMapData/loadGameState already not
		// catching: different callers want to handle a failed load
		// differently (a full-page error today; an inline form error once
		// this is called from order-building UI).
		refresh: async () => {
			const view = await loadGameState(gameId);
			gameState.set(view);
			return view;
		},

		// A thin delegate to gameState.subscribe, not a separate
		// notification mechanism — gameState already calls a subscriber
		// immediately with the current value, then again on every change.
		// A second, parallel callback registry here would risk drifting
		// out of sync with it (e.g. code that calls gameState.set(...)
		// directly wouldn't be seen by a separately-tracked list).
		onUpdate: gameState.subscribe,
	};
}
