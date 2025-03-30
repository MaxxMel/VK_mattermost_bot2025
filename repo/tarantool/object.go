package tarantool

import (
	"bot/domain"
	"encoding/json"
	"fmt"
	tt "github.com/tarantool/go-tarantool"
)

type TarantoolStorage struct {
	conn *tt.Connection
}

func NewTarantoolStorage(addr string, opts tt.Opts) (*TarantoolStorage, error) {
	conn, err := tt.Connect(addr, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Tarantool: %w", err)
	}
	return &TarantoolStorage{
		conn: conn,
	}, nil
}

func (t *TarantoolStorage) CreatePoll(id string, description string, options []domain.Option) (*domain.Poll, error) {
	if _, err := t.GetPoll(id); err == nil {
		return nil, fmt.Errorf("poll with id '%s' already exists", id)
	}

	poll := &domain.Poll{
		ID:          id,
		Description: description,
		Options:     options,
		Active:      true,
	}

	data, err := json.Marshal(poll)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal poll: %w", err)
	}

	if _, err := t.conn.Insert("polls", []interface{}{id, string(data)}); err != nil {
		return nil, fmt.Errorf("failed to insert poll into Tarantool: %w", err)
	}

	return poll, nil
}

func (t *TarantoolStorage) VotePoll(poll *domain.Poll, optionId string) error {
	found := false
	for i := range poll.Options {
		if poll.Options[i].OptID == optionId {
			poll.Options[i].Votes++
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("option with id '%s' not found in poll '%s'", optionId, poll.ID)
	}

	data, err := json.Marshal(poll)
	if err != nil {
		return fmt.Errorf("failed to marshal updated poll: %w", err)
	}

	// Обновляем опрос через Replace. Функция Replace возвращает ([]interface{}, error).
	if _, err := t.conn.Replace("polls", []interface{}{poll.ID, string(data)}); err != nil {
		return fmt.Errorf("failed to replace poll in Tarantool: %w", err)
	}

	return nil
}

func (t *TarantoolStorage) GetPollOptions(id string) (*[]domain.Option, error) {
	poll, err := t.GetPoll(id)
	if err != nil {
		return nil, err
	}
	return &poll.Options, nil
}

func (t *TarantoolStorage) DeletePoll(id string) error {
	// Функция Delete возвращает ([]interface{}, error)
	if _, err := t.conn.Delete("polls", "primary", []interface{}{id}); err != nil {
		return fmt.Errorf("failed to delete poll '%s': %w", id, err)
	}
	return nil
}

func (t *TarantoolStorage) EndPoll(id string) error {
	poll, err := t.GetPoll(id)
	if err != nil {
		return err
	}

	poll.Active = false
	data, err := json.Marshal(poll)
	if err != nil {
		return fmt.Errorf("failed to marshal poll after end: %w", err)
	}

	if _, err := t.conn.Replace("polls", []interface{}{poll.ID, string(data)}); err != nil {
		return fmt.Errorf("failed to replace poll to end it: %w", err)
	}

	return nil
}

func (t *TarantoolStorage) GetPoll(id string) (*domain.Poll, error) {
	// Выбираем кортеж по ID в первичном индексе.
	resp, err := t.conn.Select("polls", "primary", 0, 1, tt.IterEq, []interface{}{id})
	if err != nil {
		return nil, fmt.Errorf("failed to select poll '%s': %w", id, err)
	}
	if len(resp) == 0 {
		return nil, fmt.Errorf("poll with id '%s' not found", id)
	}

	// Ожидаем, что tuple имеет формат: [id, jsonString]
	tuple, ok := resp[0].([]interface{})
	if !ok || len(tuple) < 2 {
		return nil, fmt.Errorf("unexpected tuple format in Tarantool for poll '%s'", id)
	}
	jsonStr, ok := tuple[1].(string)
	if !ok {
		return nil, fmt.Errorf("failed to cast poll data to string for poll '%s'", id)
	}

	var poll domain.Poll
	if err := json.Unmarshal([]byte(jsonStr), &poll); err != nil {
		return nil, fmt.Errorf("failed to unmarshal poll from JSON: %w", err)
	}

	return &poll, nil
}
