package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
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
	FlagAddress             = "host-address"
	FlagWolBroadcastAddress = "broadcast-address"
	FlagWolBroadcastPort    = "broadcast-port"
	FlagHassURL             = "homeassistant-url"
	FlagHassWebhook         = "homeassistant-webhook-id"
	FlagAgentPort           = "daemon-port"
	FlagDaemonKey           = "daemon-key"
)

type Config struct {
	Host          HostConfig          `yaml:"host"`
	WakeOnLan     *WakeOnLanConfig    `yaml:"wake_on_lan,omitempty"`
	HomeAssistant HomeAssistantConfig `yaml:"homeassistant"`
	Grub          *GrubConfig         `yaml:"grub,omitempty"`
	Daemon        DaemonConfig        `yaml:"daemon"`
}

type DaemonConfig struct {
	Port              int    `yaml:"port"`
	APIKey            string `yaml:"api_key,omitempty"`
	ReportBootOptions bool   `yaml:"report_boot_options"`
}

type GrubConfig struct {
	ConfigPath      string `yaml:"config_path,omitempty"`
	WaitTimeSeconds int    `yaml:"wait_time_seconds,omitempty"`
	URL             string `yaml:"url,omitempty"`
}

type WakeOnLanConfig struct {
	Address string `yaml:"address,omitempty"`
	Port    int    `yaml:"port,omitempty"`
}

type HostConfig struct {
	Address    string `yaml:"address"`
	MACAddress string `yaml:"mac"`
}

type HomeAssistantConfig struct {
	URL       string `yaml:"url"`
	WebhookID string `yaml:"webhook_id"`
}

func (c *Config) Minimal() *Config {
	cp := *c
	if cp.WakeOnLan != nil {
		wol := *cp.WakeOnLan
		if wol.Address == DefaultWolBroadcastAddress {
			wol.Address = ""
		}
		if wol.Port == DefaultWolBroadcastPort {
			wol.Port = 0
		}
		if wol.Address == "" && wol.Port == 0 {
			cp.WakeOnLan = nil
		} else {
			cp.WakeOnLan = &wol
		}
	}
	if cp.Grub != nil {
		grub := *cp.Grub
		if grub.WaitTimeSeconds == DefaultGrubWaitSeconds {
			grub.WaitTimeSeconds = 0
		}
		if grub.WaitTimeSeconds == 0 && grub.ConfigPath == "" && grub.URL == "" {
			cp.Grub = nil
		} else {
			cp.Grub = &grub
		}
	}
	return &cp
}

func NewViper(cfgFile string) *viper.Viper {
	v := viper.New()
	v.SetEnvPrefix("GRUBSTATION")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.AddConfigPath(".")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}
	return v
}

func Unmarshal(v *viper.Viper) (*Config, error) {
	var cfg Config
	// Use "yaml" tags for unmarshaling to avoid redundancy
	if err := v.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "yaml"
	}); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	if cfg.Daemon.Port == 0 {
		cfg.Daemon.Port = DefaultAgentPort
	}

	// Ensure sub-structs exist if we want to apply defaults
	if cfg.Grub == nil {
		cfg.Grub = &GrubConfig{}
	}
	if cfg.Grub.WaitTimeSeconds == 0 {
		cfg.Grub.WaitTimeSeconds = DefaultGrubWaitSeconds
	}

	return &cfg, nil
}

func Save(cfg *Config, path string) error {
	out, err := yaml.Marshal(cfg.Minimal())
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, out, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

func SaveExhaustive(cfg *Config, path string) error {
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, out, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}
