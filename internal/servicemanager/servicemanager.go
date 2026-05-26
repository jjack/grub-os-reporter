package servicemanager

import (
	"errors"

	"github.com/jjack/grubstation/internal/config"
)

// Manager defines the interface for managing the agent as a background service.
type Manager interface {
	Name() string
	IsActive() bool
	IsInstalled() (bool, error)
	CheckPermissions() error
	Install(configPath string) error
	Preview(configPath string) (string, error)
	Uninstall() error
	Start() error
	Stop() error
	Configure(cfg *config.Config) error
}

var ErrNotSupported = errors.New("no supported service manager detected")
