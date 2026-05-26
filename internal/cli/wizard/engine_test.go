package wizard

import (
	"errors"
	"net"
	"testing"
)

func TestBuildIfaceOptions_Pure(t *testing.T) {
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	ifaces := []net.Interface{
		{Name: "eth0", HardwareAddr: mac},
	}
	ipProvider := func(net.Interface) ([]string, map[string]string) {
		return []string{"192.168.1.100"}, nil
	}

	opts := BuildIfaceOptions(ifaces, ipProvider)
	if len(opts) != 1 || opts[0].Label != "eth0" {
		t.Errorf("unexpected options: %v", opts)
	}
}

func TestValidatePort_Pure(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		isReinstall bool
		currentPort int
		checker     func(int) error
		wantErr     bool
	}{
		{"valid and available", "8081", false, 0, func(p int) error { return nil }, false},
		{"empty", "", false, 0, nil, true},
		{"invalid format", "abc", false, 0, nil, true},
		{"too low", "0", false, 0, nil, true},
		{"too high", "65536", false, 0, nil, true},
		{"reinstall same port", "8081", true, 8081, nil, false},
		{"in use", "8081", false, 0, func(p int) error { return errors.New("in use") }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePort(tt.port, tt.isReinstall, tt.currentPort, tt.checker)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePort() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildWolOptions(t *testing.T) {
	ips := []string{"192.168.1.50", "10.0.0.50"}
	broadcasts := map[string]string{
		"192.168.1.50": "192.168.1.255",
		"10.0.0.50":    "10.0.0.255",
	}

	opts := BuildWolOptions(ips, broadcasts)

	if len(opts) != 3 {
		t.Fatalf("expected 3 options, got %d", len(opts))
	}
	if opts[0].Value != "255.255.255.255" {
		t.Errorf("expected DefaultWolBroadcastAddress, got %s", opts[0].Value)
	}
	if opts[1].Value != "192.168.1.255" {
		t.Errorf("expected subnet broadcast 192.168.1.255, got %s", opts[1].Value)
	}
}

func TestAssembleConfig(t *testing.T) {
	cfg := AssembleConfig("eth0", 8081, "1.2.3.255", true, 2, "path")
	if cfg.Host.Interface != "eth0" || cfg.Daemon.Port != 8081 || cfg.WakeOnLan.Address != "1.2.3.255" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestCheckPortAvailability(t *testing.T) {
	t.Run("available", func(t *testing.T) {
		// Use port 0 to let the OS choose an available port
		l, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatal(err)
		}
		port := l.Addr().(*net.TCPAddr).Port
		_ = l.Close()

		err = CheckPortAvailability(port)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("unavailable", func(t *testing.T) {
		l, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = l.Close() }()
		port := l.Addr().(*net.TCPAddr).Port

		err = CheckPortAvailability(port)
		if err == nil {
			t.Error("expected error for unavailable port, got nil")
		}
	})
}
