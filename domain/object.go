package domain

import "fmt"

type Option struct {
	OptID string `json:"id"`
	Text  string `json:"text"`
	Votes int    `json:"votes"`
}

type Poll struct {
	ID          string   `json:"id"`
	Description string   `json:"question"`
	Options     []Option `json:"options"`
	Active      bool     `json:"active"`
}

func (p *Poll) Vote(optionID string) error {
	if !p.Active {
		return fmt.Errorf("голосование закрыто")
	}
	for i := range p.Options {
		if p.Options[i].OptID == optionID {
			p.Options[i].Votes++
			return nil
		}
	}
	return fmt.Errorf("вариант ответа с ID %s не найден", optionID)
}
