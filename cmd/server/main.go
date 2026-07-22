package main

import (
	"fmt"

	"github.com/matt-in-space/diplomacy/application/gameplay"
)

func main() {
	fmt.Println("Starting new Diplomacy service...")
	games := gameplay.NewMemoryGameRepository()
	players := gameplay.NewMemoryPlayerRepository()
	s := gameplay.NewService(games, players)
	_ = s
	fmt.Println("Diplomacy service running!")
}
