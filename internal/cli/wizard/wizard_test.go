package wizard

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jjack/grubstation/internal/config"
	"github.com/spf13/cobra"
	"github.com/yarlson/tap"
)

/*
// These tests hang in the CI environment because they use interactive tap components.
// We keep them here for local manual testing.

func TestGenerateConfigInteractive_Success(t *testing.T) {
	// ... (content omitted)
}

func TestGenerateConfigInteractive_Aborted(t *testing.T) {
	// ... (content omitted)
}
*/

func TestAssembleConfig_Complete(t *testing.T) {
	cfg := AssembleConfig("1.2.3.4", "mac", "wol", "http://ha", "webhook", 8081, true, 5, "/boot/grub/grub.cfg", "http://grub")
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
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	tapOut := tap.NewMockWritable()
	tap.SetTermIO(nil, tapOut)
	defer tap.SetTermIO(nil, nil)

	cfg := &config.Config{
		Host: config.HostConfig{
			Address:    "192.168.1.50",
			MACAddress: "00:11:22:33:44:55",
		},
		WakeOnLan: &config.WakeOnLanConfig{
			Address: "192.168.1.255",
			Port:    99,
		},
		HomeAssistant: config.HomeAssistantConfig{
			URL:       "http://ha.local:8123",
			WebhookID: strings.Repeat("a", 64),
		},
		Daemon: config.DaemonConfig{
			Port:              8081,
			ReportBootOptions: true,
		},
		Grub: &config.GrubConfig{
			WaitTimeSeconds: 2,
		},
	}

	PrintConfigSummary(cmd, cfg, "/etc/grubstation/config.yaml")

	out := strings.Join(tapOut.Buffer, "")
	if !strings.Contains(out, "/etc/grubstation/config.yaml") {
		t.Errorf("expected config path, got %s", out)
	}
	if !strings.Contains(out, "aaaaaaaa") {
		t.Errorf("expected webhook id, got %s", out)
	}
}
