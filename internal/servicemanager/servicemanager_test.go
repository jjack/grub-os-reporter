package servicemanager

import (
	"testing"

	"github.com/jjack/grubstation/internal/config"
)

type mockMgr struct {
	name      string
	active    bool
	installed bool
}

func (m *mockMgr) Name() string                              { return m.name }
func (m *mockMgr) IsActive() bool                            { return m.active }
func (m *mockMgr) IsInstalled() (bool, error)                { return m.installed, nil }
func (m *mockMgr) CheckPermissions() error                   { return nil }
func (m *mockMgr) Install(configPath string) error           { return nil }
func (m *mockMgr) Preview(configPath string) (string, error) { return "", nil }
func (m *mockMgr) Uninstall() error                          { return nil }
func (m *mockMgr) Start() error                              { return nil }
func (m *mockMgr) Stop() error                               { return nil }
func (m *mockMgr) Configure(cfg *config.Config) error        { return nil }

func TestRegistry(t *testing.T) {
	r := NewRegistry()

	r.Register("service1", func() Manager { return &mockMgr{name: "service1", active: true} })
	r.Register("service2", func() Manager { return &mockMgr{name: "service2", active: false} })

	t.Run("Get", func(t *testing.T) {
		m := r.Get("service1")
		if m == nil || m.Name() != "service1" {
			t.Errorf("expected service1, got %v", m)
		}

		m = r.Get("nonexistent")
		if m != nil {
			t.Errorf("expected nil, got %v", m)
		}
	})

	t.Run("Detect", func(t *testing.T) {
		m, err := r.Detect()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if m == nil || m.Name() != "service1" {
			t.Errorf("expected service1, got %v", m)
		}

		r2 := NewRegistry()
		r2.Register("service2", func() Manager { return &mockMgr{name: "service2", active: false} })
		m, err = r2.Detect()
		if err != ErrNotSupported {
			t.Errorf("expected ErrNotSupported, got %v", err)
		}
	})

	t.Run("ActiveServices", func(t *testing.T) {
		active := r.ActiveServices()
		if len(active) != 1 || active[0] != "service1" {
			t.Errorf("expected [service1], got %v", active)
		}
	})

	t.Run("SupportedServices", func(t *testing.T) {
		supported := r.SupportedServices()
		if len(supported) != 2 || supported[0] != "service1" || supported[1] != "service2" {
			t.Errorf("expected [service1, service2], got %v", supported)
		}
	})
}
