package main

import (
	"log"
	"net/http"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/core/gamemap"
	"github.com/matt-in-space/diplomacy/infrastructure/memory"
	"github.com/matt-in-space/diplomacy/web"
)

func main() {
	gr := memory.NewGameRepository()
	pr := memory.NewPlayerRepository()
	sr := memory.NewSessionRepository()

	maps := loadMaps()
	mr := memory.NewGameMapRepository(maps...)

	s := gameplay.NewGameplayService(gr, mr)
	_ = s // not wired to any handler yet — that's the game/SPA phase

	authService := auth.NewService(pr, sr)
	mux := web.NewMux(authService)

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
