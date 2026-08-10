// Circular icons for units (tank = army, ship = fleet) and supply centers
// (city skyline), anchored at each province's labelAt point (already
// present in the map JSON, unused since text labels were removed — this is
// the reuse flagged as likely back then). Split the same way every module
// here is: computeIconPlacements is pure and DOM-free (tested in
// _tests/mapIcons.test.mjs); renderIcons/updateIconScale build/update the
// actual DOM and are verified live only, matching mapRender.mjs's
// untested-DOM-construction precedent.

const SVG_NS = "http://www.w3.org/2000/svg";

// Design size in SVG user-units at the base/fit zoom level. Chosen against
// this map's real geometry (the smallest relevant provinces are ~30
// user-units across) — tunable, not exact science.
const ICON_RADIUS = 7;

// Distance from labelAt each icon is offset when a province has both a
// unit and a supply center, so neither fully occludes the other.
const COLOCATION_OFFSET = 8;

// computeIconPlacements is pure: given the already-loaded map JSON and the
// already-loaded PlayerView from gameState, returns where every icon goes
// and what it looks like. No DOM, no fetching — everything it needs is
// already in memory by the time main() calls it.
export function computeIconPlacements(mapData, view) {
	const placements = [];

	const unitsByProvince = new Map();
	for (const unit of view.Units ?? []) {
		// Dislodged units have no ProvinceID (only DislodgedFrom) — retreat-
		// phase display isn't built anywhere yet (same call made for order
		// display in application/gameplay/player_view.go), and isn't
		// reachable in the actual game yet either.
		if (!unit.ProvinceID) continue;
		unitsByProvince.set(unit.ProvinceID, unit);
	}

	for (const [provinceId, province] of Object.entries(mapData.provinces)) {
		const unit = unitsByProvince.get(provinceId);
		const isSupplyCenter = Boolean(province.supplyCenter);
		if (!unit && !isSupplyCenter) continue;

		// A missing anchor point shouldn't happen, but isn't something to
		// crash on either.
		const labelAt = province.labelAt;
		if (!labelAt) continue;
		const [labelX, labelY] = labelAt;

		const both = Boolean(unit) && isSupplyCenter;
		const unitX = both ? labelX + COLOCATION_OFFSET : labelX;
		const scX = both ? labelX - COLOCATION_OFFSET : labelX;

		if (unit) {
			placements.push({
				kind: "unit",
				provinceId,
				x: unitX,
				y: labelY,
				nationId: unit.NationID,
				unitType: unit.Type === "fleet" ? "fleet" : "army",
			});
		}
		if (isSupplyCenter) {
			placements.push({
				kind: "supply-center",
				provinceId,
				x: scX,
				y: labelY,
				nationId: view.SupplyCenterOwners?.[provinceId] ?? "",
			});
		}
	}

	return placements;
}

// renderIcons builds the actual icon elements and appends them to svg —
// two nested groups per icon, an outer one fixed at the province's
// position and an inner one holding the visible circle+symbol that gets
// counter-scaled on zoom (see updateIconScale) so icons stay a constant
// on-screen size regardless of how zoomed in the map is.
export function renderIcons(svg, placements) {
	for (const placement of placements) {
		const outer = document.createElementNS(SVG_NS, "g");
		outer.setAttribute("class", "map-icon");
		outer.setAttribute("transform", `translate(${placement.x},${placement.y})`);

		const inner = document.createElementNS(SVG_NS, "g");
		inner.setAttribute("class", "map-icon-scale");

		const nationClass = placement.nationId || "unowned";
		const circle = document.createElementNS(SVG_NS, "circle");
		circle.setAttribute("r", String(ICON_RADIUS));
		circle.setAttribute("class", `icon-circle ${placement.kind} ${nationClass}`);
		inner.appendChild(circle);
		inner.appendChild(makeSymbol(placement));

		outer.appendChild(inner);
		svg.appendChild(outer);
	}
}

// updateIconScale keeps every icon a constant on-screen size regardless of
// the map's current zoom level — called from mapViewport.mjs's
// onZoomChange hook after every pan/zoom change.
export function updateIconScale(svg, zoomFactor) {
	for (const el of svg.querySelectorAll(".map-icon-scale")) {
		el.setAttribute("transform", `scale(${1 / zoomFactor})`);
	}
}

function makeSymbol(placement) {
	if (placement.kind === "supply-center") return makeSkylineSymbol();
	return placement.unitType === "fleet" ? makeShipSymbol() : makeTankSymbol();
}

// A simple side-view tank silhouette: hull+turret as one stepped outline
// (two separately-rounded overlapping rects merged into a shapeless blob
// with no line between them — a real result of an earlier attempt, not a
// hypothetical), plus a cannon line and three wheels. Legible at small
// size is the bar, not realism. Sized well inside ICON_RADIUS so the
// colored circle stays visible as a ring around it.
function makeTankSymbol() {
	const g = document.createElementNS(SVG_NS, "g");
	g.setAttribute("class", "icon-symbol");
	g.appendChild(
		svgEl("path", {
			d: "M-3.5,2.5 L-3.5,0.5 L-1.5,0.5 L-1.5,-1.5 L2,-1.5 L2,0.5 L3.5,0.5 L3.5,2.5 Z",
		}),
	);
	g.appendChild(svgEl("line", { x1: 2, y1: -1, x2: 4.5, y2: -1 }));
	for (const cx of [-2, 0, 2]) {
		g.appendChild(svgEl("circle", { cx, cy: 2.5, r: 0.9 }));
	}
	return g;
}

// A simple hull, mast, and flag. Sized well inside ICON_RADIUS, same
// reasoning as makeTankSymbol.
function makeShipSymbol() {
	const g = document.createElementNS(SVG_NS, "g");
	g.setAttribute("class", "icon-symbol");
	g.appendChild(svgEl("path", { d: "M-4,0.5 L4,0.5 L3,3 L-3,3 Z" }));
	g.appendChild(svgEl("line", { x1: 0, y1: 0.5, x2: 0, y2: -4 }));
	g.appendChild(svgEl("path", { d: "M0,-4 L2.5,-3 L0,-2 Z" }));
	return g;
}

// Three bottom-aligned rectangles of varying height — reads as a skyline
// even very small. Sized well inside ICON_RADIUS, same reasoning as
// makeTankSymbol — this one especially needed it, its solid rectangles
// filled almost the whole circle at the original size.
function makeSkylineSymbol() {
	const g = document.createElementNS(SVG_NS, "g");
	g.setAttribute("class", "icon-symbol");
	g.appendChild(svgEl("rect", { x: -4, y: -0.5, width: 2, height: 4 }));
	g.appendChild(svgEl("rect", { x: -1, y: -2.5, width: 2, height: 6 }));
	g.appendChild(svgEl("rect", { x: 2, y: -1.5, width: 2, height: 5 }));
	return g;
}

function svgEl(tag, attrs) {
	const el = document.createElementNS(SVG_NS, tag);
	for (const [key, value] of Object.entries(attrs)) {
		el.setAttribute(key, String(value));
	}
	return el;
}
