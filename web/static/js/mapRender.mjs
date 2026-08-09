const SVG_NS = "http://www.w3.org/2000/svg";

// renderMap draws one <path> per province and one <text> label per
// province. Each path's id matches the backend's ProvinceID exactly, so
// future click handling and state-driven styling can just be
// document.getElementById(id) — no separate lookup table to keep in sync.
// (Coast targeting for fleet orders isn't drawn as a visual boundary — the
// real board doesn't show it either, it's implicit in adjudication.)
//
// Shapes are drawn in one pass and labels in a second, later pass — labels
// must not risk being painted over by a later province's fill.
export function renderMap(container, mapData) {
	const svg = document.createElementNS(SVG_NS, "svg");
	svg.setAttribute("viewBox", "-10 258 208 272");
	svg.setAttribute("class", "diplomacy-map");

	for (const [provinceId, province] of Object.entries(mapData.provinces)) {
		svg.appendChild(makeProvincePath(provinceId, province));
	}

	for (const [provinceId, province] of Object.entries(mapData.provinces)) {
		svg.appendChild(makeLabel(provinceId, province));
	}

	container.appendChild(svg);
}

function makeProvincePath(id, province) {
	const classes = ["province"];
	if (province.type === "water") classes.push("water");
	if (province.supplyCenter) classes.push("supply-center");

	const path = makePath(id, province.d, classes.join(" "));
	const title = document.createElementNS(SVG_NS, "title");
	title.textContent = province.name;
	path.appendChild(title);
	return path;
}

function makePath(id, d, className) {
	const path = document.createElementNS(SVG_NS, "path");
	path.setAttribute("id", id);
	path.setAttribute("d", d);
	path.setAttribute("class", className);
	return path;
}

// Label text is the short province id (e.g. "MAO"), not the full name —
// several provinces are too narrow to fit "Mid-Atlantic Ocean" or "English
// Channel" at a readable size. The full name is still available via the
// province path's <title>.
function makeLabel(id, province) {
	const [x, y] = province.labelAt;
	const text = document.createElementNS(SVG_NS, "text");
	text.setAttribute("x", String(x));
	text.setAttribute("y", String(y));
	text.setAttribute("class", "label");
	text.textContent = id.toUpperCase();
	return text;
}
