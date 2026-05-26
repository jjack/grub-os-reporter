package wizard

import (
	"bytes"
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
