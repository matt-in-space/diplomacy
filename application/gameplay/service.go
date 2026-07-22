package gameplay

type Service struct {
	games GameRepository
}

func NewService(games GameRepository) *Service {
	return &Service{
		games: games,
	}
}
