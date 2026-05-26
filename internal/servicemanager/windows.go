//go:build windows

package servicemanager

import (
	"github.com/jjack/grubstation/internal/config"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	WindowsServiceName = "GrubStation"
)

type WindowsService struct{}

func NewWindowsService() Manager {
	return &WindowsService{}
}

// RegisterDefaultServices registers the Windows Service manager.
func RegisterDefaultServices(r *Registry) {
	r.Register("windows-service", NewWindowsService)
}

func (w *WindowsService) Name() string {
	return "windows-service"
}

func (w *WindowsService) IsActive() bool {
	return true
}

func (w *WindowsService) IsInstalled() (bool, error) {
	h, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return false, err
	}
	m := &mgr.Mgr{Handle: h}
	defer m.Disconnect()

	s, err := m.OpenService(WindowsServiceName)
	if err == nil {
		s.Close()
		return true, nil
	}
	return false, nil
}

func (w *WindowsService) CheckPermissions() error {
	// In the Native WiX flow, the TUI doesn't need admin permissions
	// because the MSI handled the privileged tasks.
	return nil
}

func (w *WindowsService) Install(configPath string) error {
	// Service installation is handled by the WiX installer (MSI)
	return nil
}

func (w *WindowsService) Preview(configPath string) (string, error) {
	return "Service installation and configuration are handled automatically by the Windows installer (MSI).", nil
}

func (w *WindowsService) Uninstall() error {
	// Service uninstallation is handled by the WiX installer (MSI)
	return nil
}

func (w *WindowsService) Start() error {
	h, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return nil
	}
	defer windows.CloseServiceHandle(h)

	s, err := windows.OpenService(h, windows.StringToUTF16Ptr(WindowsServiceName), windows.SERVICE_START|windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return nil
	}
	defer windows.CloseServiceHandle(s)

	err = windows.StartService(s, 0, nil)
	if err != nil {
		if errno, ok := err.(windows.Errno); ok && (errno == windows.ERROR_SERVICE_ALREADY_RUNNING || errno == windows.ERROR_ACCESS_DENIED) {
			return nil
		}
		return err
	}
	return nil
}

func (w *WindowsService) Stop() error {
	h, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return nil
	}
	defer windows.CloseServiceHandle(h)

	s, err := windows.OpenService(h, windows.StringToUTF16Ptr(WindowsServiceName), windows.SERVICE_STOP|windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return nil
	}
	defer windows.CloseServiceHandle(s)

	var status windows.SERVICE_STATUS
	err = windows.ControlService(s, windows.SERVICE_CONTROL_STOP, &status)
	if err != nil {
		if errno, ok := err.(windows.Errno); ok && errno == windows.ERROR_ACCESS_DENIED {
			return nil
		}
		return err
	}

	return nil
}

func (w *WindowsService) Configure(cfg *config.Config) error {
	// Firewall configuration is handled by the WiX installer (MSI) using a program-based rule
	return nil
}
