package wizard

import (
	"bytes"
	"net"
	"strings"
	"testing"

	"github.com/jjack/grubstation/internal/config"
)

func TestPrintConfigSummary(t *testing.T) {
	var buf bytes.Buffer

	cfg := &config.Config{
		Host: config.HostConfig{
			Interface: "eth0",
			MAC:       "00:11:22:33:44:55",
		},
		WakeOnLan: config.WakeOnLanConfig{
			Address: "192.168.1.255",
			Port:    99,
		},
		Daemon: config.DaemonConfig{
			Port:              8081,
			ReportBootOptions: true,
		},
		Grub: config.GrubConfig{
			NetworkWaitTime: 2,
		},
	}

	PrintConfigSummary(&buf, cfg, "/etc/grubstation/config.yaml")

	out := buf.String()
	if !strings.Contains(out, "/etc/grubstation/config.yaml") {
		t.Errorf("expected config path, got %s", out)
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
	if len(opts) != 1 || !strings.Contains(opts[0].Key, "eth0") {
		t.Errorf("unexpected options: %v", opts)
	}
	if opts[0].Value != 0 {
		t.Errorf("expected value 0, got %v", opts[0].Value)
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
