package gamemap

import _ "embed"

//go:embed testdata/western_europe.json
var westernEuropeJSON []byte

// WesternEurope returns the bundled western-europe map — the same reduced
// fixture core/gamemap's own tests use, and currently the only map data
// shipped with the binary. It's a stand-in either way (a full Diplomacy map
// is unrelated future work); embedding it doesn't make it more or less
// "real" than it already was as the sole map in the system.
func WesternEurope() (*GameMap, error) {
	return Load(westernEuropeJSON)
}
