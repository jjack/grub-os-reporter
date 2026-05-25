package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type State struct {
	WebhookID   string `json:"webhook_id,omitempty"`
	APIKey      string `json:"api_key,omitempty"`
	HADaemonURL string `json:"ha_daemon_url,omitempty"`
	HAGrubURL   string `json:"ha_grub_url,omitempty"`
}

func LoadState(configPath string) (*State, error) {
	dir := filepath.Dir(configPath)
	statePath := filepath.Join(dir, "state.json")

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{}, nil
		}
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func SaveState(configPath string, state *State) error {
	dir := filepath.Dir(configPath)
	statePath := filepath.Join(dir, "state.json")

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0o600)
}
