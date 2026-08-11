import { loadMapData } from "./mapData.mjs";
import { renderMap } from "./mapRender.mjs";
import { attachMapViewport } from "./mapViewport.mjs";
import { createGameClient } from "./gameClient.mjs";
import { computeIconPlacements, renderIcons, updateIconScale } from "./mapIcons.mjs";
import { updateSidebar } from "./sidebarRender.mjs";

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

	const { gameId, mapId } = readMountData(mount);
	const game = createGameClient(gameId);

	mount.textContent = "Loading map…";
	try {
		const mapData = await loadMapData(mapId);

		game.onUpdate((view) => {
			// The immediate call at registration fires with whatever's
			// already in the store — null until refresh() below resolves.
			// Nothing to render yet.
			if (!view) return;

			mount.textContent = "";
			const svg = renderMap(mount, mapData);
			renderIcons(svg, computeIconPlacements(mapData, view));
			updateIconScale(svg, 1); // explicit initial scale, matching the base/fit view
			attachMapViewport(svg, { onZoomChange: (zoomFactor) => updateIconScale(svg, zoomFactor) });
		});

		// A second, independent subscription — the store already supports
		// multiple subscribers. Unlike the map-render callback above, this
		// one has no "only correct once" caveat: rebuilding list <li>s and
		// setting textContent is naturally safe to re-run on every future
		// update, not just the first.
		game.onUpdate((view) => {
			if (!view) return;
			updateSidebar(view, mapData);
		});

		await game.refresh();
	} catch (err) {
		mount.textContent = `Failed to load the game: ${err.message}`;
	}
}

// Guarded so this module stays importable from a Node test environment
// (see _tests/main.test.mjs, which imports readMountData) without trying
// to touch a nonexistent `document` as a side effect of the import itself.
if (typeof document !== "undefined") {
	main();
}
