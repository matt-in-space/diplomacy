// Pan/zoom for the map's <svg viewBox>. Split into a pure, DOM-free core
// (clampViewBox/zoomViewBox/panViewBox — unit-testable, see
// _tests/mapViewport.test.mjs) and a thin DOM-wiring layer
// (attachMapViewport), mirroring mapData.mjs/main.mjs's testable-core-
// plus-thin-shell split elsewhere in this codebase.

// Never let the view zoom out further than "the whole map fits" (base), or
// in further than this multiple of base's scale.
const MAX_ZOOM = 8;

// clampViewBox keeps box within base — never larger than base (can't zoom
// out past fit) and never positioned so it shows area outside base (can't
// pan past the map's edges).
export function clampViewBox(base, box) {
	const width = Math.min(box.width, base.width);
	const height = Math.min(box.height, base.height);
	const maxX = base.x + base.width - width;
	const maxY = base.y + base.height - height;
	return {
		width,
		height,
		x: Math.min(Math.max(box.x, base.x), maxX),
		y: Math.min(Math.max(box.y, base.y), maxY),
	};
}

// zoomViewBox scales box by factor (<1 zooms in, >1 zooms out), keeping
// focal — an {x,y} point in SVG user-space, e.g. the cursor position —
// visually stationary. The resulting zoom level is clamped to [1x,
// MAX_ZOOM] relative to base before the focal-point adjustment, so the
// clamp can't itself shift the focal point away from where it started.
export function zoomViewBox(base, box, factor, focal) {
	const minWidth = base.width / MAX_ZOOM;
	const minHeight = base.height / MAX_ZOOM;
	const width = Math.min(Math.max(box.width * factor, minWidth), base.width);
	const height = Math.min(Math.max(box.height * factor, minHeight), base.height);

	// Keep focal's relative position within the box the same before and
	// after scaling.
	const relX = (focal.x - box.x) / box.width;
	const relY = (focal.y - box.y) / box.height;

	return clampViewBox(base, {
		x: focal.x - relX * width,
		y: focal.y - relY * height,
		width,
		height,
	});
}

// panViewBox translates box by dx/dy (SVG user-space units).
export function panViewBox(base, box, dx, dy) {
	return clampViewBox(base, { ...box, x: box.x + dx, y: box.y + dy });
}

// Fraction of the current viewBox's width/height moved per arrow-key
// press, and the wheel zoom factor per notch — both tunable, nothing else
// depends on the exact values.
const KEY_PAN_FRACTION = 0.1;
const WHEEL_ZOOM_FACTOR = 1.2;

// Pointer must move at least this many CSS pixels from pointerdown before
// it's treated as a drag rather than a click — so a plain click on a
// province still reaches future click-to-select handling instead of every
// click being swallowed as a zero-distance drag.
const DRAG_THRESHOLD_PX = 4;

// attachMapViewport wires up wheel-zoom, drag-to-pan, and arrow-key-pan on
// svg, reading its current viewBox attribute (the fit-everything rect
// renderMap already computes) as the base/fit bound everything else is
// clamped against.
export function attachMapViewport(svg) {
	const base = parseViewBox(svg.getAttribute("viewBox"));
	let current = { ...base };

	function apply() {
		svg.setAttribute(
			"viewBox",
			`${current.x} ${current.y} ${current.width} ${current.height}`,
		);
	}

	function toSvgPoint(clientX, clientY) {
		const ctm = svg.getScreenCTM().inverse();
		const pt = svg.createSVGPoint();
		pt.x = clientX;
		pt.y = clientY;
		return pt.matrixTransform(ctm);
	}

	svg.setAttribute("tabindex", "0");

	svg.addEventListener(
		"wheel",
		(event) => {
			event.preventDefault();
			const focal = toSvgPoint(event.clientX, event.clientY);
			const factor = event.deltaY > 0 ? WHEEL_ZOOM_FACTOR : 1 / WHEEL_ZOOM_FACTOR;
			current = zoomViewBox(base, current, factor, focal);
			apply();
		},
		{ passive: false },
	);

	let dragging = false;
	let pointerId = null;
	let lastClientX = 0;
	let lastClientY = 0;
	let downClientX = 0;
	let downClientY = 0;

	svg.addEventListener("pointerdown", (event) => {
		pointerId = event.pointerId;
		downClientX = lastClientX = event.clientX;
		downClientY = lastClientY = event.clientY;
	});

	svg.addEventListener("pointermove", (event) => {
		if (pointerId === null || event.pointerId !== pointerId) return;

		if (!dragging) {
			const moved = Math.hypot(event.clientX - downClientX, event.clientY - downClientY);
			if (moved < DRAG_THRESHOLD_PX) return;
			dragging = true;
			svg.classList.add("dragging");
			// Best-effort: capture keeps pan tracking smooth if the pointer
			// leaves the svg mid-drag, but its absence shouldn't stop
			// dragging from working — pointermove still fires on svg via
			// event bubbling as long as the pointer stays roughly over it.
			try {
				svg.setPointerCapture(pointerId);
			} catch {
				/* no-op */
			}
		}

		// Client-pixel delta converted to SVG user-space via the current
		// viewBox's scale. Dragging right/down should reveal content to
		// the left/up, hence the negation.
		const rect = svg.getBoundingClientRect();
		const scaleX = current.width / rect.width;
		const scaleY = current.height / rect.height;
		const dx = -(event.clientX - lastClientX) * scaleX;
		const dy = -(event.clientY - lastClientY) * scaleY;
		lastClientX = event.clientX;
		lastClientY = event.clientY;

		current = panViewBox(base, current, dx, dy);
		apply();
	});

	function endDrag(event) {
		if (event.pointerId !== pointerId) return;
		if (dragging) {
			try {
				svg.releasePointerCapture(pointerId);
			} catch {
				/* no-op — see the matching try/catch around setPointerCapture */
			}
		}
		dragging = false;
		pointerId = null;
		svg.classList.remove("dragging");
	}
	svg.addEventListener("pointerup", endDrag);
	svg.addEventListener("pointercancel", endDrag);

	svg.addEventListener("keydown", (event) => {
		const step = { x: current.width * KEY_PAN_FRACTION, y: current.height * KEY_PAN_FRACTION };
		switch (event.key) {
			case "ArrowLeft":
				current = panViewBox(base, current, -step.x, 0);
				break;
			case "ArrowRight":
				current = panViewBox(base, current, step.x, 0);
				break;
			case "ArrowUp":
				current = panViewBox(base, current, 0, -step.y);
				break;
			case "ArrowDown":
				current = panViewBox(base, current, 0, step.y);
				break;
			case "0":
				current = { ...base };
				break;
			default:
				return;
		}
		event.preventDefault();
		apply();
	});
}

function parseViewBox(value) {
	const [x, y, width, height] = value.split(/\s+/).map(Number);
	return { x, y, width, height };
}
