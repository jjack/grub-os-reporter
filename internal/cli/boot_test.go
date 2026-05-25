//go:build linux

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/daemon"
	"github.com/jjack/grubstation/internal/grub"
	"github.com/jjack/grubstation/internal/homeassistant"
	"github.com/jjack/grubstation/internal/host"
	"github.com/jjack/grubstation/internal/servicemanager"
)

type mockServiceManager struct {
	name         string
	activeCalls  int
	active       bool
	installed    bool
	installErr   error
	uninstallErr error
	startErr     error
	stopErr      error
}

func (m *mockServiceManager) Name() string { return m.name }
func (m *mockServiceManager) IsActive(ctx context.Context) bool {
	res := m.active
	if m.activeCalls == 0 {
		res = true // Force true for Detect
	}
	m.activeCalls++
	return res
}
func (m *mockServiceManager) IsInstalled(ctx context.Context) (bool, error) {
	return m.installed, nil
}
func (m *mockServiceManager) CheckPermissions(ctx context.Context) error { return nil }
func (m *mockServiceManager) Install(ctx context.Context, configPath string) error {
	return m.installErr
}
func (m *mockServiceManager) Preview(ctx context.Context, configPath string) (string, error) {
	return "preview", nil
}
func (m *mockServiceManager) Uninstall(ctx context.Context) error { return m.uninstallErr }
func (m *mockServiceManager) Start(ctx context.Context) error     { return m.startErr }
func (m *mockServiceManager) Stop(ctx context.Context) error      { return m.stopErr }
func (m *mockServiceManager) Configure(ctx context.Context, cfg *config.Config) error {
	return nil
}

func TestBootListCmd(t *testing.T) {
	tempGrub := t.TempDir() + "/grub.cfg"
	_ = os.WriteFile(tempGrub, []byte("menuentry 'Ubuntu' {}\nmenuentry 'Windows' {}"), 0o644)

	deps := &CommandDeps{
		Grub: func() *grub.Grub { g := grub.NewGrub(); g.ConfigPath = tempGrub; return g }(),
	}

	cmd := NewBootListCmd(deps)
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "Available Boot Options:") {
		t.Errorf("expected header, got %q", out.String())
	}
	if !strings.Contains(out.String(), "- Ubuntu") {
		t.Errorf("expected Ubuntu, got %q", out.String())
	}
	if !strings.Contains(out.String(), "- Windows") {
		t.Errorf("expected Windows, got %q", out.String())
	}
}

func TestBootListCmd_Empty(t *testing.T) {
	tempGrub := t.TempDir() + "/grub.cfg"
	_ = os.WriteFile(tempGrub, []byte(""), 0o644)

	deps := &CommandDeps{
		Grub: func() *grub.Grub { g := grub.NewGrub(); g.ConfigPath = tempGrub; return g }(),
	}

	cmd := NewBootListCmd(deps)
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "(None found)") {
		t.Errorf("expected (None found), got %q", out.String())
	}
}

func TestBootPushCmd_Direct(t *testing.T) {
	// Mock HA server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer ts.Close()

	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.yaml")
	state := &config.State{
		HADaemonURL: ts.URL,
		WebhookID:   "fake",
	}
	_ = state.Save(cfgPath)

	tempGrub := tempDir + "/grub.cfg"
	_ = os.WriteFile(tempGrub, []byte("menuentry 'Ubuntu' {}"), 0o644)

	initReg := servicemanager.NewRegistry()
	initReg.Register("mock-init", func() servicemanager.Manager { return &mockServiceManager{name: "mock-init"} })

	deps := &CommandDeps{
		Config:     &config.Config{},
		ConfigFile: cfgPath,
		Grub:       func() *grub.Grub { g := grub.NewGrub(); g.ConfigPath = tempGrub; return g }(),
		Registry:   initReg,
	}

	// Ensure no socket exists to force direct push
	oldSocketPath := daemon.SocketPath
	daemon.SocketPath = filepath.Join(t.TempDir(), "non-existent.sock")
	defer func() { daemon.SocketPath = oldSocketPath }()

	cmd := NewBootPushCmd(deps)
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "Successfully pushed boot options to Home Assistant") {
		t.Errorf("expected success message, got %q", out.String())
	}
}

func TestBootPushCmd_Socket(t *testing.T) {
	oldSocketPath := daemon.SocketPath
	daemon.SocketPath = filepath.Join(t.TempDir(), "test.sock")
	defer func() { daemon.SocketPath = oldSocketPath }()

	// Start a dummy unix socket server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start a dummy Home Assistant server for registration
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer ts.Close()

	deps := &CommandDeps{
		Config: &config.Config{},
	}
	haClient := homeassistant.NewClient(ts.URL, "fake", nil)
	d := daemon.New(daemon.Config{ReportBootOptions: true}, daemon.Metadata{}, nil, haClient)
	go func() { _ = d.Run(ctx) }()

	// Wait for socket
	found := false
	for i := 0; i < 20; i++ {
		if _, err := os.Stat(daemon.SocketPath); err == nil {
			found = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !found {
		t.Fatal("socket was never created")
	}

	cmd := NewBootPushCmd(deps)
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "Successfully pushed boot options to Home Assistant (via running daemon)") {
		t.Errorf("expected socket success message, got %q", out.String())
	}
}

func TestBootPushCmd_Direct_WithWOL(t *testing.T) {
	// Mock HA server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer ts.Close()

	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.yaml")
	state := &config.State{
		HADaemonURL: ts.URL,
		WebhookID:   "fake",
	}
	_ = state.Save(cfgPath)

	tempGrub := tempDir + "/grub.cfg"
	_ = os.WriteFile(tempGrub, []byte("menuentry 'Ubuntu' {}"), 0o644)

	deps := &CommandDeps{
		Config: &config.Config{
			WakeOnLan: config.WakeOnLanConfig{
				Address: "192.168.1.255",
				Port:    9,
			},
		},
		ConfigFile: cfgPath,
		Grub:       func() *grub.Grub { g := grub.NewGrub(); g.ConfigPath = tempGrub; return g }(),
	}

	// Ensure no socket exists to force direct push
	oldSocketPath := daemon.SocketPath
	daemon.SocketPath = filepath.Join(t.TempDir(), "non-existent.sock")
	defer func() { daemon.SocketPath = oldSocketPath }()

	cmd := NewBootPushCmd(deps)
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "Successfully pushed boot options to Home Assistant") {
		t.Errorf("expected success message, got %q", out.String())
	}
}

func TestBootPushCmd_Direct_Error(t *testing.T) {
	t.Run("MissingConfig", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "config.yaml")
		deps := &CommandDeps{Config: &config.Config{}, ConfigFile: cfgPath}
		cmd := NewBootPushCmd(deps)
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "must be configured") {
			t.Errorf("expected missing config error, got %v", err)
		}
	})

	t.Run("GrubError", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "config.yaml")
		state := &config.State{
			HADaemonURL: "http://ha",
			WebhookID:   "id",
		}
		_ = state.Save(cfgPath)

		deps := &CommandDeps{
			Config:     &config.Config{},
			ConfigFile: cfgPath,
			Grub:       func() *grub.Grub { g := grub.NewGrub(); g.ConfigPath = "/non/existent"; return g }(),
		}
		cmd := NewBootPushCmd(deps)
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "failed to open grub config") {
			t.Errorf("expected grub error, got %v", err)
		}
	})

	t.Run("HAError", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "config.yaml")
		state := &config.State{
			HADaemonURL: ts.URL,
			WebhookID:   "id",
		}
		_ = state.Save(cfgPath)

		tempGrub := tempDir + "/grub.cfg"
		_ = os.WriteFile(tempGrub, []byte("menuentry 'OS' {}"), 0o644)

		deps := &CommandDeps{
			Config:     &config.Config{},
			ConfigFile: cfgPath,
			Grub:       func() *grub.Grub { g := grub.NewGrub(); g.ConfigPath = tempGrub; return g }(),
		}
		cmd := NewBootPushCmd(deps)
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "unexpected status code") {
			t.Errorf("expected HA error, got %v", err)
		}
	})
}

func TestServiceRemoveCmd_Unregister(t *testing.T) {
	var receivedPayload map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "api/webhook/test-webhook") {
			_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}
	}))
	defer ts.Close()

	initReg := servicemanager.NewRegistry()
	mock := &mockServiceManager{name: "mock-svc", active: true}
	initReg.Register("mock-svc", func() servicemanager.Manager { return mock })

	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.yaml")
	state := &config.State{
		HADaemonURL: ts.URL,
		WebhookID:   "test-webhook",
	}
	_ = state.Save(cfgPath)

	deps := &CommandDeps{
		Config: &config.Config{
			Host: config.HostConfig{
				Address: "1.2.3.4",
				MAC:     "AA:BB:CC:DD:EE:FF",
			},
		},
		ConfigFile: cfgPath,
		Registry:   initReg,
	}

	cmd := NewServiceRemoveCmd(deps)
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "Unregistering from Home Assistant...") {
		t.Errorf("expected unregistering message, got %q", out.String())
	}

	if receivedPayload == nil {
		t.Fatal("expected to receive payload at Home Assistant mock")
	}

	if receivedPayload["action"] != "unregister_host" {
		t.Errorf("expected action 'unregister_host', got %v", receivedPayload["action"])
	}
	if receivedPayload["mac"] != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("expected mac 'AA:BB:CC:DD:EE:FF', got %v", receivedPayload["mac"])
	}
	if receivedPayload["address"] != "1.2.3.4" {
		t.Errorf("expected address '1.2.3.4', got %v", receivedPayload["address"])
	}
}

func TestServiceRemoveCmd_UnregisterFallback(t *testing.T) {
	var receivedPayload map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer ts.Close()

	initReg := servicemanager.NewRegistry()
	mock := &mockServiceManager{name: "mock-svc", active: true}
	initReg.Register("mock-svc", func() servicemanager.Manager { return mock })

	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.yaml")
	state := &config.State{
		HADaemonURL: ts.URL,
		WebhookID:   "test-webhook",
	}
	_ = state.Save(cfgPath)

	hwAddr, _ := net.ParseMAC("00:11:22:33:44:55")

	h := host.New()
	h.NetInterfaces = func() ([]net.Interface, error) {
		return []net.Interface{{Name: "eth0", HardwareAddr: hwAddr, Flags: net.FlagUp}}, nil
	}
	h.GetAddrs = func(iface net.Interface) ([]net.Addr, error) {
		return []net.Addr{&net.IPNet{IP: net.ParseIP("192.168.1.10"), Mask: net.CIDRMask(24, 32)}}, nil
	}
	h.OsStat = func(name string) (os.FileInfo, error) {
		return nil, nil // mock device file exists
	}

	deps := &CommandDeps{
		Config:     &config.Config{},
		ConfigFile: cfgPath,
		Registry:   initReg,
		Host:       h,
	}

	cmd := NewServiceRemoveCmd(deps)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedPayload["mac"] != "00:11:22:33:44:55" {
		t.Errorf("expected detected mac '00:11:22:33:44:55', got %v", receivedPayload["mac"])
	}
	if receivedPayload["address"] != "192.168.1.10" {
		t.Errorf("expected detected address '192.168.1.10', got %v", receivedPayload["address"])
	}
}
