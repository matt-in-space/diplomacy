import { test } from "node:test";
import assert from "node:assert/strict";
import { loadGameState } from "../gameState.mjs";

test("loadGameState fetches and parses the state JSON", async () => {
	const original = globalThis.fetch;
	globalThis.fetch = async (input) => {
		assert.equal(String(input), "/games/test-game/state");
		return new Response(
			JSON.stringify({
				Turn: { Season: "spring", Phase: "accept_orders", Year: 1 },
				Units: [],
				SupplyCenterOwners: {},
				CommittedNations: null,
				YourNation: "fra",
				YourOrders: null,
			}),
			{ status: 200 },
		);
	};

	try {
		const view = await loadGameState("test-game");
		assert.equal(view.YourNation, "fra");
		assert.equal(view.Turn.Phase, "accept_orders");
	} finally {
		globalThis.fetch = original;
	}
});

test("loadGameState throws on a non-ok response", async () => {
	const original = globalThis.fetch;
	globalThis.fetch = async () => new Response("forbidden", { status: 403 });

	try {
		await assert.rejects(() => loadGameState("test-game"), /403/);
	} finally {
		globalThis.fetch = original;
	}
});
