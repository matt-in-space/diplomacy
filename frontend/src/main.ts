import { loadMapData } from "./mapData.js";
import { renderMap } from "./mapRender.js";

// readMountData is a thin, DOM-decoupled validator — it takes anything
// with a .dataset shape, not literally an HTMLElement, so it's testable
// without a real DOM. Both IDs come from the Go-rendered shell
// (web/templates/game.html), which already has them loaded server-side to
// render the page at all.
export function readMountData(mount: {
	dataset: { gameId?: string; mapId?: string };
}): { gameId: string; mapId: string } {
	const { gameId, mapId } = mount.dataset;
	if (!gameId || !mapId) {
		throw new Error("missing game/map id on the #app mount element");
	}
	return { gameId, mapId };
}

async function main(): Promise<void> {
	const mount = document.getElementById("app");
	if (!mount) {
		throw new Error("missing #app mount element");
	}

	const { mapId } = readMountData(mount);

	mount.textContent = "Loading map…";
	try {
		const mapData = await loadMapData(mapId);
		mount.textContent = "";
		renderMap(mount, mapData);
	} catch (err) {
		mount.textContent = `Failed to load the map: ${(err as Error).message}`;
	}
}

// Guarded so this module stays importable from a Node test environment
// (see main.test.ts, which imports readMountData) without trying to touch
// a nonexistent `document` as a side effect of the import itself.
if (typeof document !== "undefined") {
	main();
}
