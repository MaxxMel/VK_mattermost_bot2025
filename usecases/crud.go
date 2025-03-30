package usecases

import "bot/domain"

type CRUD interface {
	CreatePoll(id string, description string, options []domain.Option) (*domain.Poll, error)
	VotePoll(poll *domain.Poll, optionId string) error
	GetPollOptions(id string) (*[]domain.Option, error)
	EndPoll(id string) error
	DeletePoll(id string) error
	GetPoll(id string) (*domain.Poll, error)
}
