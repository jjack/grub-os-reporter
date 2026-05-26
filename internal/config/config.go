package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultWolBroadcastAddress = "255.255.255.255"
	DefaultWolBroadcastPort    = 9
	DefaultAgentPort           = 8081
	DefaultGrubWaitSeconds     = 2
)

const (
	FlagGrubConfig          = "grub-config"
	FlagMac                 = "host-mac"
	FlagInterface           = "host-interface"
	FlagWolBroadcastAddress = "broadcast-address"
	FlagWolBroadcastPort    = "broadcast-port"
	FlagHassURL             = "homeassistant-url"
	FlagHassWebhook         = "homeassistant-webhook-id"
	FlagAgentPort           = "daemon-port"
	FlagDaemonKey           = "daemon-key"
)

type Config struct {
	Host      HostConfig      `yaml:"host"`
	WakeOnLan WakeOnLanConfig `yaml:"wake_on_lan"`
	Grub      GrubConfig      `yaml:"grub"`
	Daemon    DaemonConfig    `yaml:"daemon"`
}

type DaemonConfig struct {
	Port              int    `yaml:"port"`
	APIKey            string `yaml:"api_key,omitempty"`
	ReportBootOptions bool   `yaml:"report_boot_options"`
}

type GrubConfig struct {
	Path            string `yaml:"path,omitempty"`
	NetworkWaitTime int    `yaml:"network_wait_time,omitempty"`
	URL             string `yaml:"url,omitempty"`
}

type WakeOnLanConfig struct {
	Address string `yaml:"address,omitempty"`
	Port    int    `yaml:"port,omitempty"`
}

type HostConfig struct {
	Interface string `yaml:"interface"`
	MAC       string `yaml:"mac"`
}

func (c *Config) Minimal() *Config {
	cp := *c
	if cp.WakeOnLan.Address == DefaultWolBroadcastAddress {
		cp.WakeOnLan.Address = ""
	}
	if cp.WakeOnLan.Port == DefaultWolBroadcastPort {
		cp.WakeOnLan.Port = 0
	}
	if cp.Grub.NetworkWaitTime == DefaultGrubWaitSeconds {
		cp.Grub.NetworkWaitTime = 0
	}
	return &cp
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	if cfg.Daemon.Port == 0 {
		cfg.Daemon.Port = DefaultAgentPort
	}
	if cfg.Grub.NetworkWaitTime == 0 {
		cfg.Grub.NetworkWaitTime = DefaultGrubWaitSeconds
	}

	return &cfg, nil
}

func Save(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)
	return encoder.Encode(cfg.Minimal())
}

func SaveExhaustive(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)
	return encoder.Encode(cfg)
}

func ValidateURL(s string) error {
	if s == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}
	return nil
}

func ValidatePort(s string) error {
	if s == "" {
		return fmt.Errorf("port cannot be empty")
	}
	p, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}
	if p < 1 || p > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

func ValidateGrubWaitTime(s string) error {
	if s == "" {
		return nil // Optional
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid number")
	}
	if v < 0 || v > 3600 {
		return fmt.Errorf("wait time must be between 0 and 3600 seconds")
	}
	return nil
}

func ValidateWebhookID(s string) error {
	if s == "" {
		return fmt.Errorf("webhook ID cannot be empty")
	}
	if len(s) < 6 {
		return fmt.Errorf("webhook ID is too short")
	}
	return nil
}
