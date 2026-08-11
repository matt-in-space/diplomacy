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
// A map data payload is:
//   { mapId: string, nations: { [id]: { name: string } }, provinces: { [id]: <above> } }
// nations is the authoritative roster of nations on this map — not
// derived from any particular game's current state (contrast with
// PlayerView, which only reflects what's true on the board *right now*),
// so it stays correct even after an elimination.
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
