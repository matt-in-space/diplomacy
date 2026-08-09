# Map geometry extraction

`wiki-diplomacy.svg` is [`File:Diplomacy.svg`](https://commons.wikimedia.org/wiki/File:Diplomacy.svg)
from Wikimedia Commons, by Martin Asal, dual-licensed CC BY-SA 3.0 / GFDL.
Attribution requirement: see the credit line in `web/templates/game.html`
and the `_comment` field in `web/static/maps/*.json` — both need to stay in
sync with any province geometry sourced from this file.

`extract_provinces.py` is the tool that turned this into the real
(non-placeholder) `d`/`labelAt` values in `web/static/maps/western-europe-subset.json`.
It's a one-off, hand-run tool, not part of the build — see its own
docstring for how it works, its usage, and known defects in the source
file to watch for (some provinces' fills are split across two `<path>`
elements and need manual stitching; Spain was one, found and fixed during
the western-Europe-subset extraction).

To extract more provinces (e.g. when `core/gamemap` grows beyond the
current 8-province subset): add entries to the `PROVINCES` dict at the top
of the script, run it, and hand-merge the output into the target map JSON
— every extraction needs a visual sanity check before trusting it, the
script doesn't do that for you.
