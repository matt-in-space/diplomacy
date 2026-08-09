// The static, per-map visual layer: SVG path geometry keyed by the same
// ProvinceIDs core/gamemap already uses. Deliberately separate from any
// per-game state — this is the same for every game on a given map, so it's
// fetched once and cached by the browser like any other static asset,
// never embedded per-request into the page.
//
// A province visual looks like:
//   {
//     name: string,
//     type: "inland" | "coastal" | "water",
//     supplyCenter: boolean,
//     d: string,               // SVG path data
//     labelAt: [number, number],
//   }
// A map data payload is { mapId: string, provinces: { [id]: <above> } }.
// Multi-coast provinces (e.g. Spain's spa-nc/spa-sc) aren't drawn as a
// visual boundary — the real board doesn't show one either, it's implicit
// in adjudication (core/gamemap's own coasts model).

export async function loadMapData(mapId) {
	const res = await fetch(`/static/maps/${mapId}.json`);
	if (!res.ok) {
		throw new Error(`failed to load map ${mapId}: ${res.status}`);
	}
	return await res.json();
}
