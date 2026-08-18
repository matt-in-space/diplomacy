package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/application/lobby"
	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func handleNewGameForm(tmpl pages) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := tmpl.gamesNew()
		if err == nil {
			err = t.ExecuteTemplate(w, "layout", newPageData(w, r))
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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

func handleJoinGameForm(tmpl pages) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := tmpl.gamesJoin()
		if err == nil {
			err = t.ExecuteTemplate(w, "layout", newPageData(w, r))
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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

func handleGameSetupLobby(tmpl pages, lobbyService *lobby.Service, authService *auth.Service) http.HandlerFunc {
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

		t, err := tmpl.gameSetupLobby()
		if err == nil {
			err = t.ExecuteTemplate(w, "layout", data)
		}
		if err != nil {
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

func handleGame(tmpl pages, lobbyService *lobby.Service) http.HandlerFunc {
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
		t, err := tmpl.game()
		if err == nil {
			err = t.Execute(w, gamePageData{GameID: string(id), MapID: string(setup.MapID)})
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// handleGameState serves the JSON slice of an active game's state the
// requesting player is allowed to see (gameplay.PlayerView) — the initial
// hydration payload the frontend will fetch once it consumes this (not yet
// wired up; see docs/game-ui.md). Unlike handleGame/handleGameSetupLobby,
// this checks that the requester is actually a participant in this game
// before returning anything: those pages show little enough that any
// authenticated visitor viewing them is harmless, but this endpoint
// exposes real board state, so it's held to a stricter bar. That gap in
// the two page handlers is noted, not fixed, here.
//
// Takes gameplayService directly rather than going through lobbyService —
// this is game-in-progress state, not a lobby/setup concern, and doesn't
// belong tucked behind a passthrough on a service named for a different
// bounded context just because that service happens to hold one.
func handleGameState(lobbyService *lobby.Service, gameplayService *gameplay.GameplayService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := game.GameID(r.PathValue("id"))

		// Guaranteed present: this route is wrapped in requireAuthentication.
		player, _ := currentPlayer(r)

		setup, err := lobbyService.GetGameSetup(r.Context(), id)
		if err != nil {
			if errors.Is(err, lobby.ErrGameSetupNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !slices.Contains(setup.PlayerIDs, player.ID) {
			http.Error(w, "not a participant in this game", http.StatusForbidden)
			return
		}

		status, err := lobbyService.StatusFor(r.Context(), setup)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if status != lobby.StatusActive {
			http.Error(w, "game has not started", http.StatusConflict)
			return
		}

		view, err := gameplayService.GetPlayerView(r.Context(), id, player.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(view); err != nil {
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
