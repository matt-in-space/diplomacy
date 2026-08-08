package web

import (
	"context"
	"fmt"
	"net/http"

	"github.com/matt-in-space/diplomacy/application/lobby"
	"github.com/matt-in-space/diplomacy/core/game"
)

func handleHome(lobbyService *lobby.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := homePageData{pageData: newPageData(w, r)}
		if data.CurrentPlayer != nil {
			data.Games = homeGameRows(r.Context(), lobbyService, data.CurrentPlayer.ID)
		}
		if err := homeTemplate.ExecuteTemplate(w, "layout", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// homeGameRows resolves the player's setups into ready-to-render rows. A
// listing failure degrades to an empty list rather than failing the whole
// home page — same reasoning as lobbyPlayerRows falling back per-row
// instead of erroring the page.
func homeGameRows(ctx context.Context, lobbyService *lobby.Service, playerID game.PlayerID) []homeGameRow {
	setups, err := lobbyService.ListGameSetupsForPlayer(ctx, playerID)
	if err != nil {
		return nil
	}

	rows := make([]homeGameRow, 0, len(setups))
	for _, setup := range setups {
		status, err := lobbyService.StatusFor(ctx, setup)
		if err != nil {
			continue
		}

		id := string(setup.ID)
		row := homeGameRow{GameID: id, Label: "Game " + id[:8]}
		switch status {
		case lobby.StatusPending:
			row.Href = "/games/" + id + "/lobby"
			row.Status = "Lobby, waiting for players"
		case lobby.StatusCancelled:
			row.Href = "/games/" + id + "/lobby"
			row.Status = "Cancelled"
		case lobby.StatusActive:
			row.Href = "/games/" + id
			if stored, err := lobbyService.GetGame(ctx, setup.ID); err == nil {
				row.Status = formatTurn(stored.Game.Turn)
			} else {
				row.Status = "In progress"
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// formatTurn renders a turn as "Spring 1, awaiting orders" — the phases a
// stored Game actually rests in between player actions (ProcessGame's loop
// always advances straight through the transient Resolve*/UpdateOwnership
// phases before saving). The transient-phase fallback exists only so
// nothing looks broken in the unlikely case one is ever observed at rest.
func formatTurn(t game.Turn) string {
	season := "Spring"
	if t.Season == game.Fall {
		season = "Fall"
	}
	if t.Phase == game.Completed {
		return fmt.Sprintf("%s %d, game over", season, t.Year)
	}

	phase := "processing…"
	switch t.Phase {
	case game.AcceptOrders:
		phase = "awaiting orders"
	case game.AcceptRetreats:
		phase = "awaiting retreats"
	case game.AcceptAdjustments:
		phase = "awaiting adjustments"
	}
	return fmt.Sprintf("%s %d, %s", season, t.Year, phase)
}
