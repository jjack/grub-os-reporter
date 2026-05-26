package wizard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/grub"
	"github.com/rs/zerolog/log"
	"github.com/yarlson/tap"
	"gopkg.in/yaml.v3"
)

var (
	RunGenerateSurvey func(isReinstall bool, currentPort int, isDryRun bool, getIPInfo func(net.Interface) ([]string, map[string]string)) (*config.Config, error) = generateConfigInteractive

	ErrAborted = errors.New("setup aborted")
)

func generateConfigInteractive(isReinstall bool, currentPort int, isDryRun bool, getIPInfo func(net.Interface) ([]string, map[string]string)) (*config.Config, error) {
	// 0. Preflight check
	if err := stepConfirmOverwrite(isReinstall, isDryRun); err != nil {
		return nil, err
	}

	// 1. Detect GRUB and prompt for boot reporting
	g := grub.NewGrub()
	grubConfigPath, _ := g.DiscoverConfigPath()
	var reportsBoot bool
	if grubConfigPath != "" {
		reportsBoot = stepPromptReportBootOptions(grubConfigPath)
	}

	// 2. Network Interface
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list network interfaces: %w", err)
	}
	// Filter interfaces (up, not loopback, has MAC)
	var filtered []net.Interface
	for _, inf := range interfaces {
		if inf.Flags&net.FlagUp != 0 && inf.Flags&net.FlagLoopback == 0 && len(inf.HardwareAddr) > 0 {
			filtered = append(filtered, inf)
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no suitable network interfaces found")
	}

	selectedIface, err := stepSelectNetworkInterface(filtered, getIPInfo)
	if err != nil {
		return nil, err
	}

	// 3. Daemon Port
	agentPort, err := stepSelectDaemonPort(isReinstall, currentPort)
	if err != nil {
		return nil, err
	}

	// 4. WOL Address
	wolBroadcastAddress := stepSelectWOLAddress(selectedIface, getIPInfo)

	// 5. GRUB Wait Time
	var grubWaitTime int
	if reportsBoot {
		grubWaitTime, err = stepSelectGRUBWaitTime()
		if err != nil {
			return nil, err
		}
	}

	cfg := AssembleConfig(selectedIface.Name, agentPort, wolBroadcastAddress, reportsBoot, grubWaitTime, grubConfigPath)
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
		Message:      fmt.Sprintf("Enable remote boot selection? (Detected GRUB at %s)", grubPath),
		InitialValue: true,
	})
	log.Debug().Interface("reportsBoot", reportsBoot).Msg("User selected boot reporting preference")
	return reportsBoot
}

func stepSelectNetworkInterface(interfaces []net.Interface, getIPInfo func(net.Interface) ([]string, map[string]string)) (net.Interface, error) {
	ifaceIdx := tap.Select(context.Background(), tap.SelectOptions[int]{
		Message: "Select network interface to use",
		Options: BuildIfaceOptions(interfaces, getIPInfo),
	})
	selectedIface := interfaces[ifaceIdx]
	log.Debug().Interface("interface", selectedIface.Name).Interface("mac", selectedIface.HardwareAddr.String()).Msg("Selected network interface")
	return selectedIface, nil
}

func stepSelectDaemonPort(isReinstall bool, currentPort int) (int, error) {
	defaultValue := strconv.Itoa(config.DefaultAgentPort)
	if currentPort > 0 {
		defaultValue = strconv.Itoa(currentPort)
	}
	portStr := tap.Text(context.Background(), tap.TextOptions{
		Message:      fmt.Sprintf("Daemon Port (default: %d)", config.DefaultAgentPort),
		DefaultValue: defaultValue,
		InitialValue: defaultValue,
		Validate: func(s string) error {
			if s == "" {
				return fmt.Errorf("port cannot be empty")
			}
			p, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("invalid port: %w", err)
			}
			if p < 1 || p > 65535 {
				return fmt.Errorf("port must be between 1 and 65535")
			}
			return nil
		},
	})
	port, _ := strconv.Atoi(portStr)
	log.Debug().Interface("port", port).Msg("Selected daemon port")
	return port, nil
}

func stepSelectWOLAddress(iface net.Interface, getIPInfo func(net.Interface) ([]string, map[string]string)) string {
	ips, broadcasts := getIPInfo(iface)
	wolBroadcastAddress := tap.Select(context.Background(), tap.SelectOptions[string]{
		Message: "WOL Broadcast Address (you may need to choose subnet broadcast for cross-VLAN setups)",
		Options: BuildWolOptions(ips, broadcasts),
	})
	log.Debug().Interface("address", wolBroadcastAddress).Msg("Selected WOL broadcast address")
	return wolBroadcastAddress
}

func stepSelectGRUBWaitTime() (int, error) {
	defaultWait := strconv.Itoa(config.DefaultGrubWaitSeconds)
	waitStr := tap.Text(context.Background(), tap.TextOptions{
		Message:      "GRUB Network Wait (seconds to wait for network during boot)",
		DefaultValue: defaultWait,
		InitialValue: defaultWait,
		Validate: func(s string) error {
			return config.ValidateGrubWaitTime(s)
		},
	})
	grubWaitTime, _ := strconv.Atoi(waitStr)
	log.Debug().Interface("seconds", grubWaitTime).Msg("Selected GRUB wait time")
	return grubWaitTime, nil
}

// AssembleConfig is a pure function that populates the Config struct with wizard choices.
func AssembleConfig(ifaceName string, agentPort int, wolAddress string, reportsBoot bool, grubWait int, grubPath string) *config.Config {
	return &config.Config{
		Host: config.HostConfig{
			Interface: ifaceName,
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
