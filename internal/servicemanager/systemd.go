//go:build linux

package servicemanager

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/jjack/grubstation/internal/assets"
	"github.com/jjack/grubstation/internal/config"
)

const systemdName = "systemd"

var systemdDir = "/run/systemd/system"

type Systemd struct {
	ServicePath  string
	OsExecutable func() (string, error)
	OsWriteFile  func(name string, data []byte, perm os.FileMode) error
	OsRemove     func(name string) error
	OsGetuid     func() int
	ExecCommand  func(ctx context.Context, name string, arg ...string) *exec.Cmd
	GetTemplate  func(name string) (*template.Template, error)
}

func NewSystemd() Manager {
	return &Systemd{
		ServicePath:  "/etc/systemd/system/grubstation.service",
		OsExecutable: os.Executable,
		OsWriteFile:  os.WriteFile,
		OsRemove:     os.Remove,
		OsGetuid:     os.Getuid,
		ExecCommand:  exec.CommandContext,
		GetTemplate:  assets.GetTemplate,
	}
}

// RegisterDefaultServices registers systemd as the service manager on Linux.
func RegisterDefaultServices(r *Registry) {
	r.Register(systemdName, NewSystemd)
}

func (s *Systemd) Name() string {
	return systemdName
}

func (s *Systemd) IsActive(ctx context.Context) bool {
	fi, err := os.Stat(systemdDir)
	return err == nil && fi.IsDir()
}

func (s *Systemd) IsInstalled(ctx context.Context) (bool, error) {
	_, err := os.Stat(s.ServicePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *Systemd) CheckPermissions(ctx context.Context) error {
	if s.OsGetuid() != 0 {
		return fmt.Errorf("this operation requires root privileges (try running with sudo)")
	}
	return nil
}

func (s *Systemd) Install(ctx context.Context, configPath string) error {
	content, err := s.Preview(ctx, configPath)
	if err != nil {
		return err
	}

	if err := s.OsWriteFile(s.ServicePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write systemd service file (are you running as root?): %w", err)
	}

	if out, err := s.ExecCommand(ctx, "systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %s", string(out))
	}

	if out, err := s.ExecCommand(ctx, "systemctl", "enable", "grubstation.service").CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable systemd service: %s", string(out))
	}

	return nil
}

func (s *Systemd) Preview(ctx context.Context, configPath string) (string, error) {
	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute config path: %w", err)
	}

	execPath, err := s.OsExecutable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	tmpl, err := s.GetTemplate("grubstation.service.tmpl")
	if err != nil {
		return "", fmt.Errorf("failed to parse systemd template: %w", err)
	}

	data := struct {
		ExecPath   string
		ConfigPath string
	}{
		ExecPath:   execPath,
		ConfigPath: absConfig,
	}

	var content strings.Builder
	if err := tmpl.Execute(&content, data); err != nil {
		return "", fmt.Errorf("failed to execute systemd template: %w", err)
	}

	return content.String(), nil
}

func (s *Systemd) Uninstall(ctx context.Context) error {
	_ = s.ExecCommand(ctx, "systemctl", "stop", "grubstation.service").Run()
	_ = s.ExecCommand(ctx, "systemctl", "disable", "grubstation.service").Run()

	if err := s.OsRemove(s.ServicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove systemd service file: %w", err)
	}

	if out, err := s.ExecCommand(ctx, "systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %s", string(out))
	}

	return nil
}

func (s *Systemd) Start(ctx context.Context) error {
	if out, err := s.ExecCommand(ctx, "systemctl", "start", "grubstation.service").CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start systemd service: %s", string(out))
	}
	return nil
}

func (s *Systemd) Stop(ctx context.Context) error {
	if out, err := s.ExecCommand(ctx, "systemctl", "stop", "grubstation.service").CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop systemd service: %s", string(out))
	}
	return nil
}

func (s *Systemd) Configure(ctx context.Context, cfg *config.Config) error {
	return nil
}
