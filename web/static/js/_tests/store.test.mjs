import { test } from "node:test";
import assert from "node:assert/strict";
import { createStore } from "../store.mjs";

test("get returns the initial value before any set", () => {
	const store = createStore(1);
	assert.equal(store.get(), 1);
});

test("set updates the value returned by get", () => {
	const store = createStore(1);
	store.set(2);
	assert.equal(store.get(), 2);
});

test("subscribe calls the subscriber immediately with the current value", () => {
	const store = createStore("initial");
	const seen = [];
	store.subscribe((value) => seen.push(value));
	assert.deepEqual(seen, ["initial"]);
});

test("subscribe calls the subscriber again on every future set", () => {
	const store = createStore(0);
	const seen = [];
	store.subscribe((value) => seen.push(value));
	store.set(1);
	store.set(2);
	assert.deepEqual(seen, [0, 1, 2]);
});

test("multiple subscribers all receive updates", () => {
	const store = createStore(0);
	const a = [];
	const b = [];
	store.subscribe((value) => a.push(value));
	store.subscribe((value) => b.push(value));
	store.set(1);
	assert.deepEqual(a, [0, 1]);
	assert.deepEqual(b, [0, 1]);
});

test("the function returned by subscribe stops further notifications", () => {
	const store = createStore(0);
	const seen = [];
	const unsubscribe = store.subscribe((value) => seen.push(value));
	unsubscribe();
	store.set(1);
	assert.deepEqual(seen, [0]);
});
