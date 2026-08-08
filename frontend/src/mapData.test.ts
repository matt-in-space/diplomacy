import { test } from "node:test";
import assert from "node:assert/strict";
import { loadMapData } from "./mapData.js";

test("loadMapData fetches and parses the map JSON", async () => {
	const original = globalThis.fetch;
	globalThis.fetch = (async (input: RequestInfo | URL) => {
		assert.equal(String(input), "/static/maps/test-map.json");
		return new Response(
			JSON.stringify({ mapId: "test-map", provinces: { par: { d: "M0,0Z", labelAt: [0, 0] } } }),
			{ status: 200 },
		);
	}) as typeof fetch;

	try {
		const data = await loadMapData("test-map");
		assert.equal(data.mapId, "test-map");
		assert.equal(data.provinces.par?.d, "M0,0Z");
	} finally {
		globalThis.fetch = original;
	}
});

test("loadMapData throws on a non-ok response", async () => {
	const original = globalThis.fetch;
	globalThis.fetch = (async () => new Response("not found", { status: 404 })) as typeof fetch;

	try {
		await assert.rejects(() => loadMapData("missing-map"), /404/);
	} finally {
		globalThis.fetch = original;
	}
});
