import { loadMapData } from "./mapData.mjs";
import { renderMap } from "./mapRender.mjs";
import { attachMapViewport } from "./mapViewport.mjs";

// readMountData is a thin, DOM-decoupled validator — it takes anything
// with a .dataset shape, not literally an HTMLElement, so it's testable
// without a real DOM. Both IDs come from the Go-rendered shell
// (web/templates/game.html), which already has them loaded server-side to
// render the page at all.
export function readMountData(mount) {
	const { gameId, mapId } = mount.dataset;
	if (!gameId || !mapId) {
		throw new Error("missing game/map id on the #app mount element");
	}
	return { gameId, mapId };
}

async function main() {
	const mount = document.getElementById("app");
	if (!mount) {
		throw new Error("missing #app mount element");
	}

	const { mapId } = readMountData(mount);

	mount.textContent = "Loading map…";
	try {
		const mapData = await loadMapData(mapId);
		mount.textContent = "";
		const svg = renderMap(mount, mapData);
		attachMapViewport(svg);
	} catch (err) {
		mount.textContent = `Failed to load the map: ${err.message}`;
	}
}

// Guarded so this module stays importable from a Node test environment
// (see _tests/main.test.mjs, which imports readMountData) without trying
// to touch a nonexistent `document` as a side effect of the import itself.
if (typeof document !== "undefined") {
	main();
}
