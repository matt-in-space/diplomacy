package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/application/lobby"
	"github.com/matt-in-space/diplomacy/core/gamemap"
	"github.com/matt-in-space/diplomacy/infrastructure/memory"
	"github.com/matt-in-space/diplomacy/web"
)

func main() {
	seed := flag.Bool("seed", false, "create fixed development users (user1@example.com, user2@example.com — password: password) and a pending western-europe-subset game (user1 hosting, user2 joined) on startup")
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

	mux := web.NewMux(authService, lobbyService, gameplayService)

	const addr = ":8080"
	log.Printf("Diplomacy server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func loadMaps() []*gamemap.GameMap {
	gm, err := gamemap.WesternEurope()
	if err != nil {
		panic(err)
	}
	return []*gamemap.GameMap{gm}
}
