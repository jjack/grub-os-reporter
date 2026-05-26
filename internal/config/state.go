package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type State struct {
	Paired      bool   `json:"paired"`
	WebhookID   string `json:"webhook_id,omitempty"`
	APIKey      string `json:"api_key,omitempty"`
	HADaemonURL string `json:"ha_daemon_url,omitempty"`
	HAGrubURL   string `json:"ha_grub_url,omitempty"`
}

func LoadState(path string) (*State, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{Paired: false}, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var state State
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *State) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(s)
}
