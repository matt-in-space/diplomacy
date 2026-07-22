package main

import (
	"fmt"
	"os"

	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func main() {
	fmt.Println("Starting new Diplomacy service...")
	gr := gameplay.NewMemoryGameRepository()
	pr := gameplay.NewMemoryPlayerRepository()

	maps := loadMaps()
	mr := gameplay.NewMemoryGameMapRepository(maps...)

	s := gameplay.NewGameplayService(gr, pr, mr)
	_ = s
	fmt.Println("Diplomacy service running!")
}

func loadMaps() []*gamemap.GameMap {
	data, err := os.ReadFile("../../core/gamemap/testdata/western_europe.json")
	if err != nil {
		panic(err)
	}

	gm, err := gamemap.Load(data)
	if err != nil {
		panic(err)
	}

	return []*gamemap.GameMap{gm}
}
