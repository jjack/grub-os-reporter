package wizard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/homeassistant"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/yarlson/tap"
	"gopkg.in/yaml.v3"
)

type SystemState struct {
	Hostname       string
	Interfaces     []net.Interface
	GrubConfigPath string
	IsReinstall    bool
	CurrentPort    int
}

var (
	RunGenerateSurvey func(context.Context, SystemState, bool, func(net.Interface) ([]string, map[string]string), func(string, *net.Interface) string, func(context.Context) ([]homeassistant.ServiceInstance, error)) (*config.Config, error) = generateConfigInteractive

	ErrAborted = errors.New("setup aborted")
)

func generateConfigInteractive(ctx context.Context, state SystemState, isDryRun bool, getIPInfo func(net.Interface) ([]string, map[string]string), getFQDN func(string, *net.Interface) string, discoverHA func(context.Context) ([]homeassistant.ServiceInstance, error)) (*config.Config, error) {
	if err := stepConfirmOverwrite(ctx, state.IsReinstall, isDryRun); err != nil {
		return nil, err
	}

	// 1. Boot Reporting (only if GRUB is present)
	var reportsBoot bool
	var err error
	if state.GrubConfigPath != "" {
		reportsBoot, err = stepPromptReportBootOptions(ctx, state.GrubConfigPath)
		if err != nil {
			return nil, err
		}
	}
	runsDaemon := true // Always daemon

	// 2. Network Interface
	selectedIface, err := stepSelectNetworkInterface(ctx, state.Interfaces, getIPInfo)
	if err != nil {
		return nil, err
	}

	// 3. Daemon Port
	agentPort, err := stepSelectDaemonPort(ctx, state, runsDaemon)
	if err != nil {
		return nil, err
	}

	// 4. WOL Address
	wolBroadcastAddress, err := stepSelectWOLAddress(ctx, selectedIface, getIPInfo)
	if err != nil {
		return nil, err
	}

	// 5. GRUB Wait Time
	grubWaitTime, finalGrubConfigPath, err := stepSelectGRUBWaitTime(ctx, state.GrubConfigPath, reportsBoot)
	if err != nil {
		return nil, err
	}

	cfg := AssembleConfig(selectedIface.Name, selectedIface.HardwareAddr.String(), wolBroadcastAddress, agentPort, reportsBoot, grubWaitTime, finalGrubConfigPath)
	return cfg, nil
}

func stepConfirmOverwrite(ctx context.Context, isReinstall, isDryRun bool) error {
	if isReinstall && !isDryRun {
		overwrite := tap.Confirm(ctx, tap.ConfirmOptions{
			Message:      "GrubStation is already configured. Do you want to re-run setup and overwrite the existing configuration?",
			InitialValue: false,
		})
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !overwrite {
			return ErrAborted
		}
	}
	return nil
}

func stepPromptReportBootOptions(ctx context.Context, grubPath string) (bool, error) {
	reportsBoot := tap.Confirm(ctx, tap.ConfirmOptions{
		Message:      "Enable remote boot selection (report GRUB options to Home Assistant)?",
		InitialValue: true,
	})
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	log.Debug().Interface("reportsBoot", reportsBoot).Msg("User selected boot reporting preference")
	return reportsBoot, nil
}

func stepSelectNetworkInterface(ctx context.Context, interfaces []net.Interface, getIPInfo func(net.Interface) ([]string, map[string]string)) (net.Interface, error) {
	ifaceIdx := tap.Select(ctx, tap.SelectOptions[int]{
		Message: "Available Network Interface",
		Options: BuildIfaceOptions(interfaces, getIPInfo),
	})
	if ctx.Err() != nil {
		return net.Interface{}, ctx.Err()
	}
	selectedIface := interfaces[ifaceIdx]
	log.Debug().Interface("interface", selectedIface.Name).Interface("mac", selectedIface.HardwareAddr.String()).Msg("Selected network interface")
	return selectedIface, nil
}

func stepSelectDaemonPort(ctx context.Context, state SystemState, runsDaemon bool) (int, error) {
	if !runsDaemon {
		return 0, nil
	}

	defaultValue := strconv.Itoa(config.DefaultAgentPort)
	if state.CurrentPort > 0 {
		defaultValue = strconv.Itoa(state.CurrentPort)
	}
	portStr := tap.Text(ctx, tap.TextOptions{
		Message:      fmt.Sprintf("Daemon Port (default: %d)", config.DefaultAgentPort),
		DefaultValue: defaultValue,
		InitialValue: defaultValue,
		Validate: func(s string) error {
			portChecker := CheckPortAvailability
			if os.Getenv("GRUBSTATION_SKIP_PORT_CHECK") == "true" {
				portChecker = func(int) error { return nil }
			}
			return ValidatePort(s, state.IsReinstall, state.CurrentPort, portChecker)
		},
	})
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}
	port, _ := strconv.Atoi(portStr)
	log.Debug().Interface("port", port).Msg("Selected daemon port")
	return port, nil
}

func stepSelectWOLAddress(ctx context.Context, iface net.Interface, getIPInfo func(net.Interface) ([]string, map[string]string)) (string, error) {
	ips, broadcasts := getIPInfo(iface)
	wolBroadcastAddress := tap.Select(ctx, tap.SelectOptions[string]{
		Message: "WOL Broadcast Address (you may need to choose subnet broadcast for cross-VLAN setups)",
		Options: BuildWolOptions(ips, broadcasts),
	})
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	log.Debug().Interface("address", wolBroadcastAddress).Msg("Selected WOL broadcast address")
	return wolBroadcastAddress, nil
}

func stepSelectGRUBWaitTime(ctx context.Context, grubConfigPath string, reportsBoot bool) (int, string, error) {
	if !reportsBoot {
		return 0, "", nil
	}

	defaultWait := strconv.Itoa(config.DefaultGrubWaitSeconds)
	waitStr := tap.Text(ctx, tap.TextOptions{
		Message:      "GRUB Network Wait (seconds to wait for network before getting next boot option from Home Assistant)",
		DefaultValue: defaultWait,
		InitialValue: defaultWait,
		Validate: func(s string) error {
			return config.ValidateGrubWaitTime(s)
		},
	})
	if ctx.Err() != nil {
		return 0, "", ctx.Err()
	}
	grubWaitTime, _ := strconv.Atoi(waitStr)
	log.Debug().Interface("seconds", grubWaitTime).Msg("Selected GRUB wait time")
	return grubWaitTime, grubConfigPath, nil
}

// AssembleConfig is a pure function that populates the Config struct.
func AssembleConfig(ifaceName string, mac string, wolAddress string, agentPort int, reportsBoot bool, grubWait int, grubPath string) *config.Config {
	cfg := &config.Config{
		Host: config.HostConfig{
			Interface: ifaceName,
			MAC:       mac,
		},
		WakeOnLan: config.WakeOnLanConfig{
			Address: wolAddress,
		},
		Daemon: config.DaemonConfig{
			Port:              agentPort,
			ReportBootOptions: reportsBoot,
		},
		Grub: config.GrubConfig{
			NetworkWaitTime: grubWait,
			Path:            grubPath,
		},
	}

	return cfg
}

func PrintConfigSummary(cmd *cobra.Command, cfg *config.Config, cfgPath string) {
	out, err := yaml.Marshal(cfg.Minimal())
	if err != nil {
		tap.Message(fmt.Sprintf("Error generating summary: %v", err))
		return
	}

	tap.Message(fmt.Sprintf("Configuration saved to %s", cfgPath))
	outStr := fmt.Sprintf("\n---\n%s", string(out))
	tap.Box(outStr, " Configuration Preview ", tap.BoxOptions{
		ContentPadding: 2,
	})
}
