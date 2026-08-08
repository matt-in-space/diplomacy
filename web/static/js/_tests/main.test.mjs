import { test } from "node:test";
import assert from "node:assert/strict";
import { readMountData } from "../main.mjs";

test("readMountData returns both ids when present", () => {
	const mount = { dataset: { gameId: "game-1", mapId: "western-europe-subset" } };
	assert.deepEqual(readMountData(mount), { gameId: "game-1", mapId: "western-europe-subset" });
});

test("readMountData throws when gameId is missing", () => {
	const mount = { dataset: { mapId: "western-europe-subset" } };
	assert.throws(() => readMountData(mount));
});

test("readMountData throws when mapId is missing", () => {
	const mount = { dataset: { gameId: "game-1" } };
	assert.throws(() => readMountData(mount));
});
