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
	seed := flag.Bool("seed", false, "create fixed development users (user1@example.com, user2@example.com — password: password) on startup")
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
		seedDevUsers(context.Background(), authService)
	}

	mux := web.NewMux(authService, lobbyService)

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
