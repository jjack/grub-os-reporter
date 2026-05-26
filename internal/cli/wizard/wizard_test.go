package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/jjack/grubstation/internal/config"
	"github.com/spf13/cobra"
	"github.com/yarlson/tap"
)

func TestAssembleConfig_Complete(t *testing.T) {
	cfg := AssembleConfig("1.2.3.4", "mac", "wol", 8081, true, 5, "/boot/grub/grub.cfg")
	if cfg.Host.Address != "1.2.3.4" {
		t.Errorf("expected address 1.2.3.4, got %s", cfg.Host.Address)
	}
	if cfg.Grub.URL != "http://grub" {
		t.Errorf("expected grub url http://grub, got %s", cfg.Grub.URL)
	}
}

func TestStepConfirmOverwrite_DryRun(t *testing.T) {
	// Dry run should not ask for confirmation
	err := stepConfirmOverwrite(context.Background(), true, true)
	if err != nil {
		t.Errorf("expected no error in dry run, got %v", err)
	}
}

func TestPrintConfigSummary(t *testing.T) {
	cmd := &cobra.Command{}

	tapOut := tap.NewMockWritable()
	tap.SetTermIO(nil, tapOut)
	defer tap.SetTermIO(nil, nil)

	cfg := &config.Config{
		Host: config.HostConfig{
			Address: "192.168.1.50",
			MAC:     "00:11:22:33:44:55",
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

	PrintConfigSummary(cmd, cfg, "/etc/grubstation/config.yaml")

	out := strings.Join(tapOut.Buffer, "")
	if !strings.Contains(out, "/etc/grubstation/config.yaml") {
		t.Errorf("expected config path, got %s", out)
	}
}
