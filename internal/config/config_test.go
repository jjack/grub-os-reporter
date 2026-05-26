package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_SaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.yaml")

	cfg := &Config{
		Host: HostConfig{
			Interface: "eth0",
			MAC:       "aa:bb:cc:dd:ee:ff",
		},
		Daemon: DaemonConfig{
			Port:              9000,
			ReportBootOptions: true,
		},
		Grub: GrubConfig{
			Path:            "/boot/grub/grub.cfg",
			NetworkWaitTime: 10,
		},
	}

	if err := Save(cfg, cfgPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.Host.MAC != cfg.Host.MAC || loaded.Daemon.Port != cfg.Daemon.Port || loaded.Grub.NetworkWaitTime != cfg.Grub.NetworkWaitTime {
		t.Errorf("loaded config does not match saved config: %+v", loaded)
	}
}

func TestConfig_LoadDefaults(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.yaml")

	// Empty yaml
	if err := os.WriteFile(cfgPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Daemon.Port != DefaultAgentPort {
		t.Errorf("expected default port %d, got %d", DefaultAgentPort, cfg.Daemon.Port)
	}
	if cfg.Grub.NetworkWaitTime != DefaultGrubWaitSeconds {
		t.Errorf("expected default wait time %d, got %d", DefaultGrubWaitSeconds, cfg.Grub.NetworkWaitTime)
	}
}

func TestConfig_Minimal(t *testing.T) {
	cfg := Config{
		Grub: GrubConfig{
			NetworkWaitTime: DefaultGrubWaitSeconds,
		},
		WakeOnLan: WakeOnLanConfig{
			Address: DefaultWolBroadcastAddress,
			Port:    DefaultWolBroadcastPort,
		},
	}
	minimal := cfg.Minimal()
	if minimal.Grub.NetworkWaitTime != 0 {
		t.Error("expected grub wait time to be 0 (omitted) in minimal config")
	}
	if minimal.WakeOnLan.Address != "" {
		t.Error("expected wake_on_lan address to be empty (omitted) in minimal config")
	}
}

func TestState_SaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	state := &State{
		Paired:      true,
		WebhookID:   "webhook123",
		APIKey:      "key456",
		HADaemonURL: "http://ha:8123",
		HAGrubURL:   "http://ha:8081",
	}

	if err := state.Save(configPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadState(configPath)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if !loaded.Paired || loaded.WebhookID != state.WebhookID || loaded.APIKey != state.APIKey {
		t.Errorf("loaded state does not match saved state: %+v", loaded)
	}
}
