import type { MapData } from "./mapData.js";

const SVG_NS = "http://www.w3.org/2000/svg";

// renderMap draws one <path> per province (plus one per coast, for
// multi-coast provinces). The outer province shape is drawn even when
// coasts exist — it's still the thing an army order or whole-territory
// ownership coloring targets; the coast sub-paths layer on top for the
// finer-grained targeting a fleet order needs. Each path's id matches the
// backend's ProvinceID/coast ID exactly, so future click handling and
// state-driven styling can just be document.getElementById(id) — no
// separate lookup table to keep in sync.
export function renderMap(container: Element, mapData: MapData): void {
	const svg = document.createElementNS(SVG_NS, "svg");
	svg.setAttribute("viewBox", "0 0 610 560");
	svg.setAttribute("class", "diplomacy-map");

	for (const [provinceId, province] of Object.entries(mapData.provinces)) {
		svg.appendChild(makePath(provinceId, province.d, "province"));
		for (const [coastId, coast] of Object.entries(province.coasts ?? {})) {
			svg.appendChild(makePath(coastId, coast.d, "province coast"));
		}
	}

	container.appendChild(svg);
}

function makePath(id: string, d: string, className: string): SVGPathElement {
	const path = document.createElementNS(SVG_NS, "path");
	path.setAttribute("id", id);
	path.setAttribute("d", d);
	path.setAttribute("class", className);
	return path;
}
