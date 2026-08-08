import { test } from "node:test";
import assert from "node:assert/strict";
import { loadMapData } from "../mapData.mjs";

test("loadMapData fetches and parses the map JSON", async () => {
	const original = globalThis.fetch;
	globalThis.fetch = async (input) => {
		assert.equal(String(input), "/static/maps/test-map.json");
		return new Response(
			JSON.stringify({
				mapId: "test-map",
				provinces: {
					par: { name: "Paris", type: "inland", supplyCenter: true, d: "M0,0Z", labelAt: [0, 0] },
				},
			}),
			{ status: 200 },
		);
	};

	try {
		const data = await loadMapData("test-map");
		assert.equal(data.mapId, "test-map");
		assert.equal(data.provinces.par?.d, "M0,0Z");
		assert.equal(data.provinces.par?.name, "Paris");
		assert.equal(data.provinces.par?.type, "inland");
		assert.equal(data.provinces.par?.supplyCenter, true);
	} finally {
		globalThis.fetch = original;
	}
});

test("loadMapData throws on a non-ok response", async () => {
	const original = globalThis.fetch;
	globalThis.fetch = async () => new Response("not found", { status: 404 });

	try {
		await assert.rejects(() => loadMapData("missing-map"), /404/);
	} finally {
		globalThis.fetch = original;
	}
});
