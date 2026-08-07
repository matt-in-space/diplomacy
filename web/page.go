package web

import (
	"net/http"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/application/lobby"
)

// pageData is what every rendered page needs: who's logged in, and any
// pending flash message.
type pageData struct {
	CurrentPlayer *auth.Player
	Flash         *flashData
}

type flashData struct {
	Kind    string
	Message string
}

// newPageData builds the common page data, popping any pending flash
// message so it only ever renders once.
func newPageData(w http.ResponseWriter, r *http.Request) pageData {
	data := pageData{}
	if player, ok := currentPlayer(r); ok {
		data.CurrentPlayer = player
	}
	if kind, message, ok := popFlash(w, r); ok {
		data.Flash = &flashData{Kind: kind, Message: message}
	}
	return data
}

// authFormPageData is what the login/signup forms need beyond the common
// fields: the validated `next` path to carry through the form submission
// (and the cross-link to the other auth form), so a gated route that
// redirected here can send the visitor back where they started.
type authFormPageData struct {
	pageData
	Next string
}

// gameSetupLobbyPageData is what the lobby view needs beyond the common
// fields: the setup itself, its computed status, and a resolved,
// ready-to-render list of participants. Status is passed separately rather
// than re-derived in the template, since computing it requires a repository
// read (Service.StatusFor) that only the handler can do. Players is
// pre-resolved for the same reason — lobby.Service deliberately doesn't
// depend on auth.PlayerRepository (looking up a display name is a
// rendering concern, not lobby logic), so turning a PlayerID into a name
// happens here in the handler, not in the template.
type gameSetupLobbyPageData struct {
	pageData
	Setup   *lobby.GameSetup
	Status  lobby.Status
	Players []lobbyPlayerRow
	// ReadyToStart and Capacity are only populated while Status is
	// StatusPending — the Start button only exists in that state, so
	// there's nothing to compute them for otherwise.
	ReadyToStart bool
	Capacity     int
}

type lobbyPlayerRow struct {
	DisplayName string
	IsHost      bool
	IsYou       bool
}
