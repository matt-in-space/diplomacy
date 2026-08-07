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
	gameSetupLobbyTemplate = parsePage("templates/game_setup_lobby.html")
)
