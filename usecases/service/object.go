package service

import (
	"bot/domain"
	re "bot/repo/ramStorage"
	// tt "bot/repo/tarantool"
)

type Service struct {
	Repo *re.RamStorage
}

func NewService(Repo *re.RamStorage) *Service {
	return &Service{
		Repo: Repo,
	}
}

/*
type Service struct {
	Repo *tt.TarantoolStorage
}

func NewService(Repo *tt.TarantoolStorage) *Service {
	return &Service{
		Repo: Repo,
	}
}
*/

func (a *Service) CreatePoll(id string, description string, options []domain.Option) (*domain.Poll, error) {
	return a.Repo.CreatePoll(id, description, options)
}

func (a *Service) GetPollOptions(id string) (*[]domain.Option, error) {
	return a.Repo.GetPollOptions(id)
}

func (a *Service) VotePoll(poll *domain.Poll, optionId string) error {
	return a.Repo.VotePoll(poll, optionId)
}

func (a *Service) EndPoll(id string) error {
	return a.Repo.EndPoll(id)
}
func (a *Service) DeletePoll(id string) error {
	return a.Repo.DeletePoll(id)
}

func (a *Service) GetPoll(id string) (*domain.Poll, error) {
	return a.Repo.GetPoll(id)
}
