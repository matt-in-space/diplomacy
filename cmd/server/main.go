package main

import (
	"log"
	"net/http"

	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/core/gamemap"
	"github.com/matt-in-space/diplomacy/infrastructure/memory"
	"github.com/matt-in-space/diplomacy/web"
)

func main() {
	gr := memory.NewGameRepository()
	pr := memory.NewPlayerRepository()  // satisfies auth.PlayerRepository
	sr := memory.NewSessionRepository() // satisfies auth.SessionRepository

	maps := loadMaps()
	mr := memory.NewGameMapRepository(maps...)

	s := gameplay.NewGameplayService(gr, mr)
	_ = s  // not wired to any handler yet — that's the auth/gameplay phase
	_ = pr // not wired to any handler yet — that's the auth phase
	_ = sr // not wired to any handler yet — that's the auth phase

	mux := web.NewMux()

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
