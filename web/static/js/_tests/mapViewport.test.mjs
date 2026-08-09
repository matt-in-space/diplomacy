import { test } from "node:test";
import assert from "node:assert/strict";
import { clampViewBox, zoomViewBox, panViewBox } from "../mapViewport.mjs";

const base = { x: 0, y: 0, width: 100, height: 100 };

test("clampViewBox leaves a box already inside base unchanged", () => {
	const box = { x: 10, y: 10, width: 50, height: 50 };
	assert.deepEqual(clampViewBox(base, box), box);
});

test("clampViewBox pulls a box back from the bottom-right edge", () => {
	const box = { x: 60, y: 60, width: 50, height: 50 };
	assert.deepEqual(clampViewBox(base, box), { x: 50, y: 50, width: 50, height: 50 });
});

test("clampViewBox pulls a box back from the top-left edge", () => {
	const box = { x: -20, y: -20, width: 50, height: 50 };
	assert.deepEqual(clampViewBox(base, box), { x: 0, y: 0, width: 50, height: 50 });
});

test("clampViewBox shrinks a box larger than base down to base's size", () => {
	const box = { x: 0, y: 0, width: 150, height: 150 };
	assert.deepEqual(clampViewBox(base, box), { x: 0, y: 0, width: 100, height: 100 });
});

test("zoomViewBox zooming in shrinks the box and preserves the focal point's relative position", () => {
	const focal = { x: 50, y: 50 }; // center of `base`
	const result = zoomViewBox(base, base, 0.5, focal);
	assert.deepEqual(result, { x: 25, y: 25, width: 50, height: 50 });
	// focal's relative position within the box: 0.5 before, 0.5 after.
	assert.equal((focal.x - result.x) / result.width, 0.5);
	assert.equal((focal.y - result.y) / result.height, 0.5);
});

test("zoomViewBox can't zoom out further than base", () => {
	const result = zoomViewBox(base, base, 2, { x: 50, y: 50 });
	assert.deepEqual(result, base);
});

test("zoomViewBox clamps at MAX_ZOOM (8x) when zooming in aggressively", () => {
	const result = zoomViewBox(base, base, 0.01, { x: 50, y: 50 });
	assert.equal(result.width, base.width / 8);
	assert.equal(result.height, base.height / 8);
});

test("panViewBox translates by exactly dx/dy when within bounds", () => {
	const box = { x: 10, y: 10, width: 50, height: 50 };
	assert.deepEqual(panViewBox(base, box, 5, -5), { x: 15, y: 5, width: 50, height: 50 });
});

test("panViewBox clamps a pan that would cross an edge", () => {
	const box = { x: 40, y: 40, width: 50, height: 50 };
	assert.deepEqual(panViewBox(base, box, 20, 20), { x: 50, y: 50, width: 50, height: 50 });
});
