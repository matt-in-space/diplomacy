package gamemap

import _ "embed"

//go:embed testdata/western_europe.json
var westernEuropeJSON []byte

// WesternEuropeMapID is the ID embedded in western_europe.json — named here
// so callers that need the ID as a value (a web form's <option value>, for
// instance) reference this constant instead of duplicating the string
// independently of the fixture that actually defines it.
const WesternEuropeMapID MapID = "western-europe-subset"

// WesternEurope returns the bundled western-europe map — the same reduced
// fixture core/gamemap's own tests use, and currently the only map data
// shipped with the binary. It's a stand-in either way (a full Diplomacy map
// is unrelated future work); embedding it doesn't make it more or less
// "real" than it already was as the sole map in the system.
func WesternEurope() (*GameMap, error) {
	return Load(westernEuropeJSON)
}
