import { test } from "node:test";
import assert from "node:assert/strict";
import { computeIconPlacements } from "../mapIcons.mjs";

const mapData = {
	provinces: {
		lon: { name: "London", supplyCenter: true, labelAt: [160, 280] },
		par: { name: "Paris", supplyCenter: true, labelAt: [155, 358] },
		eng: { name: "English Channel", supplyCenter: false, labelAt: [135, 306] },
	},
};

test("a supply-center-only province gets exactly one placement, centered at labelAt", () => {
	const view = { Units: [], SupplyCenterOwners: { par: "fra" } };
	const placements = computeIconPlacements(mapData, view).filter((p) => p.provinceId === "par");

	assert.equal(placements.length, 1);
	assert.equal(placements[0].kind, "supply-center");
	assert.equal(placements[0].nationId, "fra");
	assert.deepEqual([placements[0].x, placements[0].y], mapData.provinces.par.labelAt);
});

test("a unit-only province gets exactly one placement, centered at labelAt", () => {
	const view = {
		Units: [{ ID: "u1", NationID: "eng", ProvinceID: "eng", Type: "fleet", Dislodged: false }],
		SupplyCenterOwners: {},
	};
	const placements = computeIconPlacements(mapData, view).filter((p) => p.provinceId === "eng");

	assert.equal(placements.length, 1);
	assert.equal(placements[0].kind, "unit");
	assert.equal(placements[0].unitType, "fleet");
	assert.equal(placements[0].nationId, "eng");
	assert.deepEqual([placements[0].x, placements[0].y], mapData.provinces.eng.labelAt);
});

test("a province with both a unit and a supply center (London) gets two offset placements", () => {
	const view = {
		Units: [{ ID: "u1", NationID: "eng", ProvinceID: "lon", Type: "fleet", Dislodged: false }],
		SupplyCenterOwners: { lon: "eng" },
	};
	const placements = computeIconPlacements(mapData, view).filter((p) => p.provinceId === "lon");

	assert.equal(placements.length, 2);
	const [labelX, labelY] = mapData.provinces.lon.labelAt;
	const unit = placements.find((p) => p.kind === "unit");
	const sc = placements.find((p) => p.kind === "supply-center");

	// Both offset from the shared anchor, not sitting exactly on top of it...
	assert.notEqual(unit.x, labelX);
	assert.notEqual(sc.x, labelX);
	// ...and not sitting exactly on top of each other either.
	assert.notEqual(unit.x, sc.x);
	assert.equal(unit.y, labelY);
	assert.equal(sc.y, labelY);
});

test("an unowned supply center still produces a placement, with an empty nationId", () => {
	const view = { Units: [], SupplyCenterOwners: { par: "" } };
	const placements = computeIconPlacements(mapData, view).filter((p) => p.provinceId === "par");

	assert.equal(placements.length, 1);
	assert.equal(placements[0].nationId, "");
});

test("a dislodged unit produces no placement", () => {
	const view = {
		Units: [{ ID: "u1", NationID: "eng", ProvinceID: "", DislodgedFrom: "eng", Type: "fleet", Dislodged: true }],
		SupplyCenterOwners: {},
	};
	const placements = computeIconPlacements(mapData, view).filter((p) => p.kind === "unit");

	assert.equal(placements.length, 0);
});
