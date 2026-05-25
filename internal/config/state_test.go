package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestState_SaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	state := &State{
		WebhookID:   "webhook123",
		APIKey:      "key456",
		HADaemonURL: "http://ha:8123",
		HAGrubURL:   "http://ha:8081",
	}

	// 1. Save
	if err := SaveState(configPath, state); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Verify file exists
	statePath := filepath.Join(tempDir, "state.json")
	if _, err := os.Stat(statePath); err != nil {
		t.Errorf("state.json was not created: %v", err)
	}

	// 2. Load
	loaded, err := LoadState(configPath)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if loaded.WebhookID != state.WebhookID || loaded.APIKey != state.APIKey ||
		loaded.HADaemonURL != state.HADaemonURL || loaded.HAGrubURL != state.HAGrubURL {
		t.Errorf("loaded state does not match saved state: %+v", loaded)
	}
}

func TestLoadState_NoFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	state, err := LoadState(configPath)
	if err != nil {
		t.Fatalf("LoadState failed for non-existent file: %v", err)
	}

	if state.WebhookID != "" {
		t.Errorf("expected empty state, got %+v", state)
	}
}
