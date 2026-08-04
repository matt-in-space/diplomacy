package gameplay

type GameplayService struct {
	games GameRepository
	maps  GameMapRepository
}

func NewGameplayService(games GameRepository, maps GameMapRepository) *GameplayService {
	return &GameplayService{
		games: games,
		maps:  maps,
	}
}
