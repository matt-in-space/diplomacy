package main

import (
	"fmt"

	"github.com/matt-in-space/diplomacy/application/gameplay"
)

func main() {
	fmt.Println("Starting new Diplomacy service...")
	games := gameplay.NewMemoryGameRepository()
	s := gameplay.NewService(games)
	_ = s
	fmt.Println("Diplomacy service running!")
}
