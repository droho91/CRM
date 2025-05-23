package server

import (
	"encoding/json"
	"os"
	"sync"

	"CRM/models"
)

const playersFile = "data/players.json"

var mu sync.Mutex // protect file access

func loadPlayers() (map[string]*models.PlayerState, error) {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.Open(playersFile)
	if os.IsNotExist(err) {
		return map[string]*models.PlayerState{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var list []*models.PlayerState
	if err := json.NewDecoder(file).Decode(&list); err != nil {
		return nil, err
	}
	out := make(map[string]*models.PlayerState)
	for _, p := range list {
		out[p.Username] = p
	}
	return out, nil
}

func savePlayers(players map[string]*models.PlayerState) error {
	mu.Lock()
	defer mu.Unlock()

	list := make([]*models.PlayerState, 0, len(players))
	for _, p := range players {
		list = append(list, p)
	}
	tmp := playersFile + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err = json.NewEncoder(f).Encode(list); err != nil {
		return err
	}
	f.Close()
	return os.Rename(tmp, playersFile)
}
