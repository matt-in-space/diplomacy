package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/application/lobby"
	"github.com/matt-in-space/diplomacy/core/gamemap"
	"github.com/matt-in-space/diplomacy/infrastructure/memory"
	"github.com/matt-in-space/diplomacy/web"
)

func main() {
	seed := flag.Bool("seed", false, "create fixed development users (user1@example.com, user2@example.com — password: password) and a pending western-europe-subset game (user1 hosting, user2 joined) on startup")
	dev := flag.Bool("dev", false, "read templates and static assets from web/ on disk per request instead of from the embedded copies, so edits show up on refresh without a restart (development only; run from the repository root)")
	flag.Parse()

	gr := memory.NewGameRepository()
	pr := memory.NewPlayerRepository()
	sr := memory.NewSessionRepository()
	gsr := memory.NewGameSetupRepository()

	maps := loadMaps()
	mr := memory.NewGameMapRepository(maps...)

	gameplayService := gameplay.NewGameplayService(gr, mr)
	lobbyService := lobby.NewService(gsr, gr, mr, gameplayService)

	authService := auth.NewService(pr, sr)
	if *seed {
		players := seedDevUsers(context.Background(), authService)
		seedDevGame(context.Background(), lobbyService, players)
	}

	var opts []web.Option
	if *dev {
		dir := devSourceDir()
		log.Printf("dev mode: serving templates and static assets from ./%s on disk", dir)
		opts = append(opts, web.WithSourceDir(dir))
	}
	mux := web.NewMux(authService, lobbyService, gameplayService, opts...)

	const addr = ":8080"
	log.Printf("Diplomacy server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// devSourceDir returns the web/ directory -dev reads templates and static
// assets from, or exits explaining why it can't. The path is relative to
// the working directory, so a server started from anywhere but the
// repository root would otherwise "work" and then 404 every stylesheet and
// fail every page render, with nothing on stdout to say why.
func devSourceDir() string {
	const dir = "web"
	for _, needed := range []string{"templates/layout.html", "static/pico.min.css"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(needed))); err != nil {
			wd, _ := os.Getwd()
			log.Fatalf("-dev reads templates and assets from ./%s, but %s/%s isn't there (working directory: %s).\n"+
				"Run the server from the repository root: go run ./cmd/server -dev", dir, dir, needed, wd)
		}
	}
	return dir
}

func loadMaps() []*gamemap.GameMap {
	gm, err := gamemap.WesternEurope()
	if err != nil {
		panic(err)
	}
	return []*gamemap.GameMap{gm}
}
