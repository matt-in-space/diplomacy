import { createStore } from "./store.js";

// GameState is just enough to prove the pipe works end to end: TypeScript
// compiles, the browser resolves the native ES module import, the mount
// point is found, and the game ID (read from the URL itself, not handed
// down by the Go-rendered shell) makes it onto the page. Real game state
// (units, orders, provinces) is future work, once there's a REST
// API/WebSocket protocol to model it against.
interface GameState {
	gameId: string;
}

// gameIdFromPath extracts the ID from a /games/{id} path — the single
// source of truth for which game this is, rather than duplicating it into
// the DOM via a data attribute the server would have to keep in sync.
function gameIdFromPath(pathname: string): string {
	const match = /^\/games\/([^/]+)$/.exec(pathname);
	if (!match) {
		throw new Error(`unexpected path for the game screen: ${pathname}`);
	}
	return match[1];
}

function main(): void {
	const mount = document.getElementById("app");
	if (!mount) {
		throw new Error("missing #app mount element");
	}

	const state = createStore<GameState>({ gameId: gameIdFromPath(window.location.pathname) });
	state.subscribe((s) => {
		mount.textContent = `Game ${s.gameId}`;
	});
}

main();
