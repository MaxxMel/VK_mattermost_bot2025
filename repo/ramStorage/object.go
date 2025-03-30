package ramStorage

import (
	"bot/domain"
	"fmt"
	"sync"
)

type RamStorage struct {
	mu    sync.Mutex
	polls map[string]*domain.Poll
}

func NewRamStorage() *RamStorage {
	return &RamStorage{
		polls: make(map[string]*domain.Poll),
	}
}
func (rs *RamStorage) GetPoll(id string) (*domain.Poll, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	poll, exists := rs.polls[id]
	if !exists {
		return nil, fmt.Errorf("голосование с id %s не найдено", id)
	}

	return poll, nil
}

func (rs *RamStorage) CreatePoll(id string, description string, options []domain.Option) (*domain.Poll, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if _, exists := rs.polls[id]; exists {
		return nil, fmt.Errorf("голосование с id %s уже существует", id)
	}

	poll := &domain.Poll{
		ID:          id,
		Description: description,
		Options:     options,
		Active:      true,
		// Добавьте другие поля, например, дату создания, идентификатор создателя и т.д.
	}

	rs.polls[id] = poll
	return poll, nil
}

func (rs *RamStorage) GetPollOptions(id string) (*[]domain.Option, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	poll, exists := rs.polls[id]
	if !exists {
		return nil, fmt.Errorf("голосование с id %s не найдено", id)
	}

	return &poll.Options, nil
}

func (rs *RamStorage) VotePoll(poll *domain.Poll, optionId string) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if _, exists := rs.polls[poll.ID]; !exists {
		return fmt.Errorf("голосование с id %s не найдено", poll.ID)
	}

	err := rs.polls[poll.ID].Vote(optionId)

	if err != nil {
		return fmt.Errorf("не удалось добавить голос: %w", err)
	}
	//...

	return nil
}

func (rs *RamStorage) EndPoll(id string) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	poll, exists := rs.polls[id]
	if !exists {
		return fmt.Errorf("голосование с id %s не найдено", id)
	}

	poll.Active = false
	return nil
}

func (rs *RamStorage) DeletePoll(id string) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if _, exists := rs.polls[id]; !exists {
		return fmt.Errorf("голосование с id %s не найдено", id)
	}

	delete(rs.polls, id)
	return nil
}

// Poll:
// ID
// Deskr
// Option : id, data, votesNum
//...
// Option: id, data, votesNum
// Active
// GetPoll -> Poll.Options -> из каждой опции достаем Votes
