import { test } from "node:test";
import assert from "node:assert/strict";
import { formatTurnText, buildNationRows, buildUnitRows } from "../sidebarRender.mjs";

const mapData = {
	nations: {
		eng: { name: "England" },
		fra: { name: "France" },
	},
	provinces: {
		par: { name: "Paris" },
		bre: { name: "Brest" },
	},
};

test("formatTurnText renders a normal phase", () => {
	const turn = { Season: "spring", Phase: "accept_orders", Year: 1 };
	assert.deepEqual(formatTurnText(turn), { turnLine: "Spring 1", phaseLine: "Awaiting Orders" });
});

test("formatTurnText renders the completed phase as game over", () => {
	const turn = { Season: "fall", Phase: "completed", Year: 3 };
	assert.deepEqual(formatTurnText(turn), { turnLine: "Fall 3", phaseLine: "Game Over" });
});

test("buildNationRows lists every nation in mapData, with names", () => {
	const rows = buildNationRows(mapData);
	assert.deepEqual(
		rows.sort((a, b) => a.nationId.localeCompare(b.nationId)),
		[
			{ nationId: "eng", name: "England" },
			{ nationId: "fra", name: "France" },
		],
	);
});

test("buildUnitRows only includes YourNation's units", () => {
	const view = {
		YourNation: "fra",
		Units: [
			{ ID: "fra-u", NationID: "fra", ProvinceID: "par", Type: "army" },
			{ ID: "eng-u", NationID: "eng", ProvinceID: "bre", Type: "fleet" },
		],
		YourOrders: [],
	};
	const rows = buildUnitRows(mapData, view);

	assert.equal(rows.length, 1);
	assert.equal(rows[0].unitId, "fra-u");
	assert.equal(rows[0].label, "A Paris");
});

test("buildUnitRows excludes dislodged units", () => {
	const view = {
		YourNation: "fra",
		Units: [{ ID: "fra-u", NationID: "fra", ProvinceID: "", DislodgedFrom: "par", Type: "army" }],
		YourOrders: [],
	};
	assert.deepEqual(buildUnitRows(mapData, view), []);
});

test("buildUnitRows uses the real order description when one exists", () => {
	const view = {
		YourNation: "fra",
		Units: [{ ID: "fra-u", NationID: "fra", ProvinceID: "par", Type: "army" }],
		YourOrders: [{ UnitID: "fra-u", Description: "Move → Gascony" }],
	};
	assert.equal(buildUnitRows(mapData, view)[0].orderText, "Move → Gascony");
});

test("buildUnitRows falls back to 'No orders yet' when there's no matching order", () => {
	const view = {
		YourNation: "fra",
		Units: [{ ID: "fra-u", NationID: "fra", ProvinceID: "par", Type: "army" }],
		YourOrders: [],
	};
	assert.equal(buildUnitRows(mapData, view)[0].orderText, "No orders yet");
});

test("buildUnitRows renders fleet shorthand with an F prefix", () => {
	const view = {
		YourNation: "fra",
		Units: [{ ID: "fra-u", NationID: "fra", ProvinceID: "bre", Type: "fleet" }],
		YourOrders: [],
	};
	assert.equal(buildUnitRows(mapData, view)[0].label, "F Brest");
});
