package web

import "html/template"

// parsePage combines the shared layout with one page's content block into
// its own isolated *template.Template. Parsing every page together into one
// shared set wouldn't work here — each page's {{define "content"}} would
// collide with every other page's in a single shared namespace, and
// whichever was parsed last would win for all of them.
func parsePage(files ...string) *template.Template {
	all := append([]string{"templates/layout.html"}, files...)
	return template.Must(template.ParseFS(templateFS, all...))
}

var (
	homeTemplate           = parsePage("templates/home.html")
	loginTemplate          = parsePage("templates/login.html")
	signupTemplate         = parsePage("templates/signup.html")
	gamesNewTemplate       = parsePage("templates/games_new.html")
	gamesJoinTemplate      = parsePage("templates/games_join.html")
	gameSetupLobbyTemplate = parsePage("templates/game_setup_lobby.html")
	// gameTemplate is parsed standalone, not through parsePage — this is
	// the one page that deliberately doesn't extend layout.html's Pico.css
	// nav chrome. The game screen owns its own styling entirely (currently
	// none — it's a placeholder mount point for the TypeScript frontend),
	// same split docs/user-experience.md draws between the account/lobby
	// flow and the in-game flow.
	gameTemplate = template.Must(template.ParseFS(templateFS, "templates/game.html"))
)
