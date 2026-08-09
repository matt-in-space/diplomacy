#!/usr/bin/env python3
"""Extract province geometry from Wikimedia Commons' File:Diplomacy.svg.

Used to build the traced (non-placeholder) province paths in
web/static/maps/*.json — see that file's own "_comment" field for the
license/attribution this data carries (CC BY-SA 3.0 / GFDL, Martin Asal),
which must be preserved wherever this output ends up.

Why this exists: the source SVG's <path> fill elements carry no IDs, but
its <text> labels do (id="Spa", id="Bre", ...) at each label's coordinate.
This flattens every path's Bezier curves into a polygon and, for each
province you list in PROVINCES below, finds the smallest province-fill
polygon that contains that label's point. "Smallest that contains it" is
what excludes larger polygons the label point also happens to sit inside
(e.g. a country outline containing several of its own provinces).

Usage:
    python3 extract_provinces.py > out.json
Then hand-merge out.json's "d"/"labelAt" values into the target
web/static/maps/*.json — this is NOT meant to overwrite that file
automatically. Every extraction needs a visual sanity check (render it,
look at it) before trusting it: see "Known defects" below.

Known defects in the source file (found while extracting the western
Europe subset — expect more of these across the full board):
  - Some provinces have their fill split across two adjacent <path>
    elements instead of one closed shape (Spain was — its second fragment
    had no label of its own and sat exactly along the missing boundary).
    A path missing a 'z' and with a large gap between its first and last
    point (see the closure check this script prints) is the tell. Fix by
    finding the adjacent unlabeled fragment whose endpoint matches the
    open path's start/end, and splicing the two 'd' strings together at
    that shared point.
  - PROVINCE_CLASSES below ({s1, s3, s4}) are this specific SVG revision's
    CSS classes for water/land fills, determined by inspecting the file's
    <style> block and cross-checking which classes the known
    lon/eng matches came back as. Re-verify if Wikimedia's file changes.
"""

import re
import sys
import json

SOURCE_SVG = "wiki-diplomacy.svg"

# Water and land fill classes in this SVG revision's <style> block — see
# "Known defects" above for how these were determined.
PROVINCE_CLASSES = {"s1", "s3", "s4"}

# Map our core/gamemap ProvinceID -> this SVG's <text id="..."> label.
# Extend this as more of the board gets modeled in core/gamemap. The full
# label list for the whole 34-province board (grep the source svg for
# `<text id=`) includes things like Nwy, Mos, Ank, Con, Smy, Vie, Rom, Ber,
# Stp, War, etc. — add entries here, one core/gamemap ProvinceID at a time,
# and re-run.
PROVINCES = {
	"lon": "Lon", "eng": "ENG", "bre": "Bre", "par": "Par",
	"gas": "Gas", "mao": "MAO", "por": "Por", "spa": "Spa",
}

TOKEN = re.compile(r"([MmLlHhVvCcSsQqTtAaZz])|(-?\d*\.?\d+(?:e-?\d+)?)")


def flatten(d, steps=10):
	"""Flatten an SVG path's 'd' attribute into a polygon (list of (x,y))."""
	toks = [(a or b) for a, b in TOKEN.findall(d)]
	pts, i, cur, start, cmd, prev_ctrl = [], 0, (0.0, 0.0), (0.0, 0.0), None, None

	def num():
		nonlocal i
		v = float(toks[i])
		i += 1
		return v

	while i < len(toks):
		if re.match(r"^[A-Za-z]$", toks[i]):
			cmd = toks[i]
			i += 1
			if cmd in "Zz":
				pts.append(start)
				cur = start
				continue
		rel = cmd.islower()
		C = cmd.upper()
		x0, y0 = cur
		if C == "M":
			x, y = num(), num()
			cur = (x0 + x, y0 + y) if rel else (x, y)
			start = cur
			pts.append(cur)
			cmd = "l" if rel else "L"
		elif C == "L":
			x, y = num(), num()
			cur = (x0 + x, y0 + y) if rel else (x, y)
			pts.append(cur)
		elif C == "H":
			x = num()
			cur = (x0 + x, y0) if rel else (x, y0)
			pts.append(cur)
		elif C == "V":
			y = num()
			cur = (x0, y0 + y) if rel else (x0, y)
			pts.append(cur)
		else:
			if C == "C":
				a, b, c, dd, x, y = (num() for _ in range(6))
				p1 = (x0 + a, y0 + b) if rel else (a, b)
				p2 = (x0 + c, y0 + dd) if rel else (c, dd)
				p3 = (x0 + x, y0 + y) if rel else (x, y)
			elif C == "S":
				c, dd, x, y = (num() for _ in range(4))
				p1 = prev_ctrl or (x0, y0)
				p2 = (x0 + c, y0 + dd) if rel else (c, dd)
				p3 = (x0 + x, y0 + y) if rel else (x, y)
			elif C == "Q":
				a, b, x, y = (num() for _ in range(4))
				q = (x0 + a, y0 + b) if rel else (a, b)
				p3 = (x0 + x, y0 + y) if rel else (x, y)
				p1 = (x0 + 2 / 3 * (q[0] - x0), y0 + 2 / 3 * (q[1] - y0))
				p2 = (p3[0] + 2 / 3 * (q[0] - p3[0]), p3[1] + 2 / 3 * (q[1] - p3[1]))
			elif C == "T":
				x, y = num(), num()
				p3 = (x0 + x, y0 + y) if rel else (x, y)
				p1 = p2 = None
			else:  # 'A' (elliptical arc) — approximated as a straight line
				for _ in range(5):
					num()
				x, y = num(), num()
				p3 = (x0 + x, y0 + y) if rel else (x, y)
				p1 = p2 = None
			if p1 is None:
				pts.append(p3)
			else:
				for t in range(1, steps + 1):
					t /= steps
					u = 1 - t
					pts.append((
						u**3 * x0 + 3 * u * u * t * p1[0] + 3 * u * t * t * p2[0] + t**3 * p3[0],
						u**3 * y0 + 3 * u * u * t * p1[1] + 3 * u * t * t * p2[1] + t**3 * p3[1],
					))
				prev_ctrl = (2 * p3[0] - p2[0], 2 * p3[1] - p2[1])
			cur = p3
	return pts


def contains(poly, pt):
	x, y = pt
	inside = False
	for i in range(len(poly)):
		x1, y1 = poly[i]
		x2, y2 = poly[(i + 1) % len(poly)]
		if (y1 > y) != (y2 > y):
			if x < (x2 - x1) * (y - y1) / (y2 - y1) + x1:
				inside = not inside
	return inside


def area(poly):
	a = 0.0
	for i in range(len(poly)):
		x1, y1 = poly[i]
		x2, y2 = poly[(i + 1) % len(poly)]
		a += x1 * y2 - x2 * y1
	return abs(a) / 2


def bbox(poly):
	xs = [p[0] for p in poly]
	ys = [p[1] for p in poly]
	return min(xs), min(ys), max(xs), max(ys)


def main():
	svg = open(SOURCE_SVG, encoding="utf-8").read()

	labels = {
		m.group(1): (float(m.group(2)), float(m.group(3)))
		for m in re.finditer(r'<text id="([^"]+)"[^>]*matrix\(1,0,0,1,([\d.-]+),([\d.-]+)\)', svg)
	}

	recs = []
	for attrs in re.findall(r"<path([^>]*)/>", svg):
		cls_match = re.search(r'class="([^"]+)"', attrs)
		d_match = re.search(r'\sd="([^"]+)"', attrs)
		if d_match:
			recs.append((cls_match.group(1) if cls_match else "", d_match.group(1)))

	polys = [(cls, d, flatten(d)) for cls, d in recs]

	out = {}
	for pid, label in PROVINCES.items():
		if label not in labels:
			print(f"WARNING: no <text id=\"{label}\"> found in {SOURCE_SVG}", file=sys.stderr)
			continue
		pt = labels[label]
		hits = sorted(
			(
				(area(p), cls, d, p)
				for cls, d, p in polys
				if cls in PROVINCE_CLASSES and len(p) > 2 and contains(p, pt)
			),
			key=lambda h: h[0],
		)
		if not hits:
			print(f"WARNING: no province-fill path contains {label}'s label point", file=sys.stderr)
			continue
		a, cls, d, poly = hits[0]
		closed = d.rstrip().endswith(("z", "Z"))
		gap = ((poly[0][0] - poly[-1][0]) ** 2 + (poly[0][1] - poly[-1][1]) ** 2) ** 0.5
		flag = "" if closed or gap < 2 else "  <-- OPEN PATH, needs manual repair (see module docstring)"
		print(
			f"{pid:4} {label:5} {cls:3} area={a:8.0f} bbox={tuple(round(v, 1) for v in bbox(poly))}{flag}",
			file=sys.stderr,
		)
		out[pid] = {"label": label, "d": d, "labelAt": [round(pt[0]), round(pt[1])]}

	json.dump(out, sys.stdout, indent=2)
	print()


if __name__ == "__main__":
	main()
