// The static, per-map visual layer: SVG path geometry keyed by the same
// ProvinceIDs core/gamemap already uses. Deliberately separate from any
// per-game state — this is the same for every game on a given map, so it's
// fetched once and cached by the browser like any other static asset,
// never embedded per-request into the page.
export interface ProvinceVisual {
	d: string;
	labelAt: [number, number];
	// Only present for multi-coast provinces (e.g. Spain's spa-nc/spa-sc).
	// A single-coast province's one coast is implicit — not worth
	// duplicating into this shape.
	coasts?: Record<string, { d: string }>;
}

export interface MapData {
	mapId: string;
	provinces: Record<string, ProvinceVisual>;
}

export async function loadMapData(mapId: string): Promise<MapData> {
	const res = await fetch(`/static/maps/${mapId}.json`);
	if (!res.ok) {
		throw new Error(`failed to load map ${mapId}: ${res.status}`);
	}
	return (await res.json()) as MapData;
}
