import { test, beforeEach } from "node:test";
import assert from "node:assert/strict";
import { createGameClient } from "../gameClient.mjs";
import { gameState } from "../gameState.mjs";

// gameState is a module-level singleton — shared across every test in
// this file, not a fresh instance per import. Without resetting it here,
// one test's leftover value could leak into the next and make results
// depend on run order.
beforeEach(() => {
	gameState.set(null);
});

function stubFetch(view) {
	const original = globalThis.fetch;
	globalThis.fetch = async () => new Response(JSON.stringify(view), { status: 200 });
	return () => {
		globalThis.fetch = original;
	};
}

test("refresh() updates gameState's value to the fetched view", async () => {
	const view = { Turn: { Season: "spring", Phase: "accept_orders", Year: 1 }, YourNation: "fra" };
	const restore = stubFetch(view);

	try {
		const game = createGameClient("test-game");
		assert.equal(gameState.get(), null);

		const returned = await game.refresh();

		assert.deepEqual(returned, view);
		assert.deepEqual(gameState.get(), view);
	} finally {
		restore();
	}
});

test("onUpdate(fn) is called immediately at registration with the current value", () => {
	gameState.set("already loaded");
	const game = createGameClient("test-game");

	const seen = [];
	game.onUpdate((value) => seen.push(value));

	assert.deepEqual(seen, ["already loaded"]);
});

test("onUpdate(fn) is called again, with the new value, after refresh() resolves", async () => {
	const view = { YourNation: "eng" };
	const restore = stubFetch(view);

	try {
		const game = createGameClient("test-game");
		const seen = [];
		game.onUpdate((value) => seen.push(value));

		await game.refresh();

		assert.deepEqual(seen, [null, view]);
	} finally {
		restore();
	}
});
