package wizard

import (
	"strings"
	"testing"

	"github.com/jjack/grubstation/internal/config"
	"github.com/yarlson/tap"
)

func TestAssembleConfig_Complete(t *testing.T) {
	cfg := AssembleConfig("eth0", 8081, "1.2.3.255", true, 5, "/boot/grub/grub.cfg")
	if cfg.Host.Interface != "eth0" {
		t.Errorf("expected interface eth0, got %s", cfg.Host.Interface)
	}
	if cfg.WakeOnLan.Address != "1.2.3.255" {
		t.Errorf("expected wol address 1.2.3.255, got %s", cfg.WakeOnLan.Address)
	}
}

func TestStepConfirmOverwrite_DryRun(t *testing.T) {
	// Dry run should not ask for confirmation
	err := stepConfirmOverwrite(true, true)
	if err != nil {
		t.Errorf("expected no error in dry run, got %v", err)
	}
}

func TestPrintConfigSummary(t *testing.T) {
	tapOut := tap.NewMockWritable()
	tap.SetTermIO(nil, tapOut)
	defer tap.SetTermIO(nil, nil)

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

	PrintConfigSummary(nil, cfg, "/etc/grubstation/config.yaml")

	out := strings.Join(tapOut.Buffer, "")
	if !strings.Contains(out, "/etc/grubstation/config.yaml") {
		t.Errorf("expected config path, got %s", out)
	}
}
