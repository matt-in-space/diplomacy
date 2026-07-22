package gameplay

type GameplayService struct {
	games   GameRepository
	players PlayerRepository
	maps    GameMapRepository
}

func NewGameplayService(games GameRepository, players PlayerRepository, maps GameMapRepository) *GameplayService {
	return &GameplayService{
		games:   games,
		players: players,
		maps:    maps,
	}
}
