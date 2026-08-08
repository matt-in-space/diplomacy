package web

import (
	"errors"
	"net/http"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/application/lobby"
	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func handleNewGameForm(w http.ResponseWriter, r *http.Request) {
	if err := gamesNewTemplate.ExecuteTemplate(w, "layout", newPageData(w, r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleCreateGame(lobbyService *lobby.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}

		// Guaranteed present: this route is wrapped in requireAuthentication.
		player, _ := currentPlayer(r)

		setup, err := lobbyService.CreateGameSetup(r.Context(), player.ID, gamemap.MapID(r.PostFormValue("map_id")))
		if err != nil {
			setFlash(w, "error", err.Error())
			http.Redirect(w, r, "/games/new", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/games/"+string(setup.ID)+"/lobby", http.StatusSeeOther)
	}
}

func handleJoinGameForm(w http.ResponseWriter, r *http.Request) {
	if err := gamesJoinTemplate.ExecuteTemplate(w, "layout", newPageData(w, r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleJoinGameSubmit(lobbyService *lobby.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}

		// Guaranteed present: this route is wrapped in requireAuthentication.
		player, _ := currentPlayer(r)

		setup, err := lobbyService.JoinGameSetup(r.Context(), r.PostFormValue("code"), player.ID)
		if err != nil {
			setFlash(w, "error", joinErrorMessage(err))
			http.Redirect(w, r, "/games/join", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/games/"+string(setup.ID)+"/lobby", http.StatusSeeOther)
	}
}

// joinErrorMessage turns JoinGameSetup's sentinel errors into copy a player
// would actually understand, rather than the raw wrapped error string —
// which for ErrGameSetupNotFound would just echo the code back, not
// explain what went wrong.
func joinErrorMessage(err error) string {
	switch {
	case errors.Is(err, lobby.ErrGameSetupNotFound):
		return "That invite code doesn't match a game."
	case errors.Is(err, lobby.ErrGameSetupFull):
		return "That game is already full."
	case errors.Is(err, lobby.ErrGameSetupNotOpen):
		return "That game has already started or been cancelled."
	default:
		return "Something went wrong. Please try again."
	}
}

func handleGameSetupLobby(lobbyService *lobby.Service, authService *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := game.GameID(r.PathValue("id"))

		setup, err := lobbyService.GetGameSetup(r.Context(), id)
		if err != nil {
			if errors.Is(err, lobby.ErrGameSetupNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		status, err := lobbyService.StatusFor(r.Context(), setup)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := gameSetupLobbyPageData{
			pageData: newPageData(w, r),
			Setup:    setup,
			Status:   status,
		}
		data.Players = lobbyPlayerRows(r, authService, setup, data.CurrentPlayer)

		if status == lobby.StatusPending {
			ready, capacity, err := lobbyService.ReadyToStart(r.Context(), setup)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			data.ReadyToStart, data.Capacity = ready, capacity
		}

		if err := gameSetupLobbyTemplate.ExecuteTemplate(w, "layout", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func handleStartGame(lobbyService *lobby.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := game.GameID(r.PathValue("id"))

		// Guaranteed present: this route is wrapped in requireAuthentication.
		player, _ := currentPlayer(r)

		if err := lobbyService.StartGame(r.Context(), id, player.ID); err != nil {
			setFlash(w, "error", startGameErrorMessage(err))
			http.Redirect(w, r, "/games/"+string(id)+"/lobby", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/games/"+string(id), http.StatusSeeOther)
	}
}

// startGameErrorMessage mirrors joinErrorMessage's shape — StartGame's
// sentinel errors become copy a host would understand, not the raw wrapped
// error string.
func startGameErrorMessage(err error) string {
	switch {
	case errors.Is(err, lobby.ErrNotHost):
		return "Only the host can start the game."
	case errors.Is(err, lobby.ErrGameSetupNotFull):
		return "Waiting for all players to join before starting."
	case errors.Is(err, lobby.ErrGameSetupNotOpen):
		return "This game has already started or been cancelled."
	default:
		return "Something went wrong. Please try again."
	}
}

func handleGame(lobbyService *lobby.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := game.GameID(r.PathValue("id"))

		setup, err := lobbyService.GetGameSetup(r.Context(), id)
		if err != nil {
			if errors.Is(err, lobby.ErrGameSetupNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		status, err := lobbyService.StatusFor(r.Context(), setup)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Pending or cancelled: nothing to show here yet — send the visitor
		// to the lobby, which already knows how to render either state.
		if status != lobby.StatusActive {
			http.Redirect(w, r, "/games/"+string(id)+"/lobby", http.StatusSeeOther)
			return
		}

		// Execute, not ExecuteTemplate — game.html has no {{define}} wrapper
		// to name, it's parsed standalone (see gameTemplate's doc comment).
		if err := gameTemplate.Execute(w, gamePageData{GameID: string(id), MapID: string(setup.MapID)}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// lobbyPlayerRows resolves each PlayerID in setup to a display name,
// falling back to the raw ID if the lookup fails — a player record
// disappearing shouldn't take the whole page down, just degrade one row.
func lobbyPlayerRows(r *http.Request, authService *auth.Service, setup *lobby.GameSetup, currentPlayer *auth.Player) []lobbyPlayerRow {
	rows := make([]lobbyPlayerRow, 0, len(setup.PlayerIDs))
	for _, playerID := range setup.PlayerIDs {
		name := string(playerID)
		if p, err := authService.GetPlayer(r.Context(), playerID); err == nil {
			name = p.DisplayName
		}
		rows = append(rows, lobbyPlayerRow{
			DisplayName: name,
			IsHost:      playerID == setup.HostID,
			IsYou:       currentPlayer != nil && playerID == currentPlayer.ID,
		})
	}
	return rows
}
