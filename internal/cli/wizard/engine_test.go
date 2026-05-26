package wizard

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetModeOptions(t *testing.T) {
	t.Run("with grub config", func(t *testing.T) {
		opts := GetModeOptions("/boot/grub/grub.cfg")
		if len(opts) != 3 {
			t.Errorf("expected 3 options, got %d", len(opts))
		}
	})

	t.Run("without grub config", func(t *testing.T) {
		opts := GetModeOptions("")
		if len(opts) != 1 {
			t.Errorf("expected 1 option, got %d", len(opts))
		}
	})
}

func TestGetModeFlags(t *testing.T) {
	tests := []struct {
		mode        string
		reportsBoot bool
		runsDaemon  bool
	}{
		{ModeDaemonBoth, true, true},
		{ModeDaemonShutdown, false, true},
		{ModeHookOnly, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			reports, runs := GetModeFlags(tt.mode)
			if reports != tt.reportsBoot || runs != tt.runsDaemon {
				t.Errorf("GetModeFlags(%s) = (%v, %v), want (%v, %v)", tt.mode, reports, runs, tt.reportsBoot, tt.runsDaemon)
			}
		})
	}
}

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

func TestBuildHostOptions(t *testing.T) {
	opts := BuildHostOptions("my-host", "my-host.global", "my-host.local", []string{"192.168.1.50"})

	if len(opts) != 4 {
		t.Fatalf("expected 4 options, got %d", len(opts))
	}
	if opts[0].Value != "my-host.local" {
		t.Errorf("expected option 0 value to be my-host.local")
	}
	if opts[1].Value != "my-host.global" {
		t.Errorf("expected option 1 value to be my-host.global")
	}
	if opts[3].Value != "192.168.1.50" {
		t.Errorf("expected option 3 value to be 192.168.1.50")
	}
}

func TestBuildHostOptions_Deduplication(t *testing.T) {
	opts := BuildHostOptions("my-host", "my-host", "my-host", []string{"192.168.1.50"})
	if len(opts) != 2 {
		t.Errorf("expected 2 options after deduplication, got %d", len(opts))
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

func TestValidateHAURL_Pure(t *testing.T) {
	t.Run("valid and reachable", func(t *testing.T) {
		err := ValidateHAURL(context.Background(), "http://localhost:8123", false, func(ctx context.Context, u string) error { return nil })
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("skip check", func(t *testing.T) {
		err := ValidateHAURL(context.Background(), "http://localhost:8123", true, func(ctx context.Context, u string) error {
			t.Error("urlChecker should not be called")
			return nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("unreachable", func(t *testing.T) {
		err := ValidateHAURL(context.Background(), "http://localhost:8123", false, func(ctx context.Context, u string) error { return errors.New("unreachable") })
		if err == nil || err.Error() != "unreachable" {
			t.Errorf("expected unreachable error, got %v", err)
		}
	})
}

func TestAssembleConfig(t *testing.T) {
	cfg := AssembleConfig("host", "mac", "wol", 8081, true, 2, "path")
	if cfg.Host.Address != "host" || cfg.Daemon.Port != 8081 || !cfg.Daemon.ReportBootOptions {
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

func TestCheckHAConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		err := CheckHAConnection(context.Background(), ts.URL)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		err := CheckHAConnection(context.Background(), "http://localhost:1")
		if err == nil {
			t.Error("expected error for unreachable URL, got nil")
		}
	})

	t.Run("invalid request", func(t *testing.T) {
		err := CheckHAConnection(context.Background(), " http://invalid")
		if err == nil {
			t.Error("expected error for invalid URL, got nil")
		}
	})
}

func TestBuildWolOptions_Extra(t *testing.T) {
	t.Run("no broadcast for ip", func(t *testing.T) {
		ips := []string{"192.168.1.50"}
		broadcasts := map[string]string{}
		opts := BuildWolOptions(ips, broadcasts)
		if len(opts) != 1 {
			t.Errorf("expected 1 option (default), got %d", len(opts))
		}
	})
}

func TestValidateHAURL_Extra(t *testing.T) {
	t.Run("invalid url format", func(t *testing.T) {
		err := ValidateHAURL(context.Background(), "not-a-url", false, nil)
		if err == nil {
			t.Error("expected error for invalid URL format, got nil")
		}
	})
}
