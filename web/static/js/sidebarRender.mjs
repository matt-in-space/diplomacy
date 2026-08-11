// Turn status, nation list, and unit list — the sidebar chrome in
// web/templates/game.html. Same split as every other module here: pure,
// testable formatting/row-building (see _tests/sidebarRender.test.mjs),
// plus a thin DOM-update shell verified live only (mirrors
// mapIcons.mjs's computeIconPlacements/renderIcons split).

// formatTurnText mirrors web/home.go's formatTurn in substance — raw
// Year, no calendar-year offset (same call already made and documented
// there), same phase-label mapping — just split into the plaque's two
// lines instead of one combined string.
export function formatTurnText(turn) {
	const season = turn.Season === "fall" ? "Fall" : "Spring";
	if (turn.Phase === "completed") {
		return { turnLine: `${season} ${turn.Year}`, phaseLine: "Game Over" };
	}

	const phases = {
		accept_orders: "Awaiting Orders",
		accept_retreats: "Awaiting Retreats",
		accept_adjustments: "Awaiting Adjustments",
	};
	return { turnLine: `${season} ${turn.Year}`, phaseLine: phases[turn.Phase] ?? "Processing…" };
}

// buildNationRows lists every nation on this map — existence/name only,
// no per-nation status (ownership, elimination, "waiting on") — that's a
// separate future concern. Sourced from mapData.nations, not derived from
// current game state, so a nation doesn't just vanish once eliminated.
export function buildNationRows(mapData) {
	return Object.entries(mapData.nations ?? {}).map(([nationId, nation]) => ({
		nationId,
		name: nation.name,
	}));
}

// buildUnitRows lists the requesting player's own units, each with its
// real Diplomacy shorthand ("A Paris") and current order description, if
// any. Dislodged units (no ProvinceID) are skipped — retreat-phase
// display isn't built anywhere yet (same call already made in
// mapIcons.mjs's computeIconPlacements and player_view.go's order
// filtering), and isn't reachable in the actual game yet either.
export function buildUnitRows(mapData, view) {
	const orderByUnit = new Map((view.YourOrders ?? []).map((o) => [o.UnitID, o.Description]));

	return (view.Units ?? [])
		.filter((unit) => unit.NationID === view.YourNation && unit.ProvinceID)
		.map((unit) => ({
			unitId: unit.ID,
			label: unitShorthand(unit, mapData),
			orderText: orderByUnit.get(unit.ID) ?? "No orders yet",
		}));
}

// unitShorthand mirrors player_view.go's Go function of the same name:
// "A"/"F" plus the unit's current province name.
function unitShorthand(unit, mapData) {
	const prefix = unit.Type === "fleet" ? "F" : "A";
	const province = mapData.provinces[unit.ProvinceID];
	return `${prefix} ${province ? province.name : unit.ProvinceID}`;
}

// updateSidebar refreshes the turn plaque and both sidebar lists from the
// given (already-loaded) view + mapData. Safe to call on every update, not
// just the first — unlike the map render, rebuilding list <li>s and
// setting textContent has no "only correct once" caveat.
export function updateSidebar(view, mapData) {
	const { turnLine, phaseLine } = formatTurnText(view.Turn);
	setText(".turn-status .turn", turnLine);
	setText(".turn-status .phase", phaseLine);

	const nationName = mapData.nations?.[view.YourNation]?.name ?? view.YourNation;
	setText(".unit-list-heading", `Your Units — ${nationName}`);

	replaceListItems(".nation-list", buildNationRows(mapData), (row) => {
		const swatch = `<span class="nation-swatch ${row.nationId}"></span>`;
		return `${swatch}${escapeHtml(row.name)}`;
	});

	replaceListItems(".unit-list", buildUnitRows(mapData, view), (row) => {
		const label = `<span class="unit-label">${escapeHtml(row.label)}</span>`;
		const order = `<span class="unit-order">${escapeHtml(row.orderText)}</span>`;
		return `${label}${order}`;
	});
}

function setText(selector, text) {
	const el = document.querySelector(selector);
	if (el) el.textContent = text;
}

function replaceListItems(selector, rows, renderRowHtml) {
	const list = document.querySelector(selector);
	if (!list) return;
	list.innerHTML = rows.map((row) => `<li>${renderRowHtml(row)}</li>`).join("");
}

function escapeHtml(value) {
	return String(value)
		.replaceAll("&", "&amp;")
		.replaceAll("<", "&lt;")
		.replaceAll(">", "&gt;");
}
