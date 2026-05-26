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
	RunGenerateSurvey func(SystemState, bool, func(net.Interface) ([]string, map[string]string), func(string, *net.Interface) string, func() ([]homeassistant.ServiceInstance, error)) (*config.Config, error) = generateConfigInteractive

	ErrAborted = errors.New("setup aborted")
)

func generateConfigInteractive(state SystemState, isDryRun bool, getIPInfo func(net.Interface) ([]string, map[string]string), getFQDN func(string, *net.Interface) string, discoverHA func() ([]homeassistant.ServiceInstance, error)) (*config.Config, error) {
	if err := stepConfirmOverwrite(state.IsReinstall, isDryRun); err != nil {
		return nil, err
	}

	// 1. Boot Reporting (only if GRUB is present)
	var reportsBoot bool
	if state.GrubConfigPath != "" {
		reportsBoot = stepPromptReportBootOptions(state.GrubConfigPath)
	}
	runsDaemon := true // Always daemon

	// 2. Network Interface
	selectedIface, err := stepSelectNetworkInterface(state.Interfaces, getIPInfo)
	if err != nil {
		return nil, err
	}

	// 3. Daemon Port
	agentPort, err := stepSelectDaemonPort(state, runsDaemon)
	if err != nil {
		return nil, err
	}

	// 4. WOL Address
	wolBroadcastAddress, err := stepSelectWOLAddress(selectedIface, getIPInfo)
	if err != nil {
		return nil, err
	}

	// 5. GRUB Wait Time
	grubWaitTime, finalGrubConfigPath, err := stepSelectGRUBWaitTime(state.GrubConfigPath, reportsBoot)
	if err != nil {
		return nil, err
	}

	cfg := AssembleConfig(selectedIface.Name, selectedIface.HardwareAddr.String(), wolBroadcastAddress, agentPort, reportsBoot, grubWaitTime, finalGrubConfigPath)
	return cfg, nil
}

func stepConfirmOverwrite(isReinstall, isDryRun bool) error {
	if isReinstall && !isDryRun {
		overwrite := tap.Confirm(context.Background(), tap.ConfirmOptions{
			Message:      "GrubStation is already configured. Do you want to re-run setup and overwrite the existing configuration?",
			InitialValue: false,
		})
		if !overwrite {
			return ErrAborted
		}
	}
	return nil
}

func stepPromptReportBootOptions(grubPath string) bool {
	reportsBoot := tap.Confirm(context.Background(), tap.ConfirmOptions{
		Message:      "Enable remote boot selection (report GRUB options to Home Assistant)?",
		InitialValue: true,
	})
	log.Debug().Interface("reportsBoot", reportsBoot).Msg("User selected boot reporting preference")
	return reportsBoot
}

func stepSelectNetworkInterface(interfaces []net.Interface, getIPInfo func(net.Interface) ([]string, map[string]string)) (net.Interface, error) {
	ifaceIdx := tap.Select(context.Background(), tap.SelectOptions[int]{
		Message: "Available Network Interface",
		Options: BuildIfaceOptions(interfaces, getIPInfo),
	})
	selectedIface := interfaces[ifaceIdx]
	log.Debug().Interface("interface", selectedIface.Name).Interface("mac", selectedIface.HardwareAddr.String()).Msg("Selected network interface")
	return selectedIface, nil
}

func stepSelectDaemonPort(state SystemState, runsDaemon bool) (int, error) {
	if !runsDaemon {
		return 0, nil
	}

	defaultValue := strconv.Itoa(config.DefaultAgentPort)
	if state.CurrentPort > 0 {
		defaultValue = strconv.Itoa(state.CurrentPort)
	}
	portStr := tap.Text(context.Background(), tap.TextOptions{
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
	port, _ := strconv.Atoi(portStr)
	log.Debug().Interface("port", port).Msg("Selected daemon port")
	return port, nil
}

func stepSelectWOLAddress(iface net.Interface, getIPInfo func(net.Interface) ([]string, map[string]string)) (string, error) {
	ips, broadcasts := getIPInfo(iface)
	wolBroadcastAddress := tap.Select(context.Background(), tap.SelectOptions[string]{
		Message: "WOL Broadcast Address (you may need to choose subnet broadcast for cross-VLAN setups)",
		Options: BuildWolOptions(ips, broadcasts),
	})
	log.Debug().Interface("address", wolBroadcastAddress).Msg("Selected WOL broadcast address")
	return wolBroadcastAddress, nil
}

func stepSelectGRUBWaitTime(grubConfigPath string, reportsBoot bool) (int, string, error) {
	if !reportsBoot {
		return 0, "", nil
	}

	defaultWait := strconv.Itoa(config.DefaultGrubWaitSeconds)
	waitStr := tap.Text(context.Background(), tap.TextOptions{
		Message:      "GRUB Network Wait (seconds to wait for network before getting next boot option from Home Assistant)",
		DefaultValue: defaultWait,
		InitialValue: defaultWait,
		Validate: func(s string) error {
			return config.ValidateGrubWaitTime(s)
		},
	})
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

func PrintConfigSummary(cmd any, cfg *config.Config, cfgPath string) {
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
