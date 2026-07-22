package gameplay

type Service struct {
	games   GameRepository
	players PlayerRepository
}

func NewService(games GameRepository, players PlayerRepository) *Service {
	return &Service{
		games:   games,
		players: players,
	}
}
