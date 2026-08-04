package web

import (
	"net/http"

	"github.com/matt-in-space/diplomacy/application/auth"
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
