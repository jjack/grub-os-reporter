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

	// Start background tasks
	fqdnChan := startFQDNResolution(ctx, state.Hostname, getFQDN)

	// 1. Installation Mode
	mode, err := stepSelectInstallationMode(ctx, state.GrubConfigPath)
	if err != nil {
		return nil, err
	}
	reportsBoot, runsDaemon := GetModeFlags(mode)

	// 2. Network Interface
	selectedIface, err := stepSelectNetworkInterface(ctx, state.Interfaces, getIPInfo)
	if err != nil {
		return nil, err
	}

	// 3. Host Address
	hostAddress, err := stepSelectHostAddress(ctx, state.Hostname, selectedIface, fqdnChan, getIPInfo, getFQDN)
	if err != nil {
		return nil, err
	}

	// 4. Daemon Port
	agentPort, err := stepSelectDaemonPort(ctx, state, runsDaemon)
	if err != nil {
		return nil, err
	}

	// 5. WOL Address
	wolBroadcastAddress, err := stepSelectWOLAddress(ctx, selectedIface, getIPInfo)
	if err != nil {
		return nil, err
	}

	// 6. GRUB Wait Time
	grubWaitTime, finalGrubConfigPath, err := stepSelectGRUBWaitTime(ctx, state.GrubConfigPath, reportsBoot)
	if err != nil {
		return nil, err
	}

	cfg := AssembleConfig(hostAddress, selectedIface.HardwareAddr.String(), wolBroadcastAddress, agentPort, reportsBoot, grubWaitTime, finalGrubConfigPath)
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

type fqdnResolutionResult struct {
	fqdn string
}

func startFQDNResolution(ctx context.Context, hostname string, getFQDN func(string, *net.Interface) string) <-chan fqdnResolutionResult {
	globalInfoChan := make(chan fqdnResolutionResult, 1)
	go func() {
		log.Debug().Interface("hostname", hostname).Msg("Starting background global FQDN resolution")
		fqdn := getFQDN(hostname, nil)
		log.Debug().Interface("fqdn", fqdn).Msg("Background global FQDN resolution complete")
		globalInfoChan <- fqdnResolutionResult{fqdn}
	}()
	return globalInfoChan
}

func stepSelectInstallationMode(ctx context.Context, grubConfigPath string) (string, error) {
	mode := tap.Select(ctx, tap.SelectOptions[string]{
		Message: "Installation Mode",
		Options: GetModeOptions(grubConfigPath),
	})
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	log.Debug().Interface("mode", mode).Msg("Selected installation mode")
	return mode, nil
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

func stepSelectHostAddress(ctx context.Context, hostname string, iface net.Interface, fqdnChan <-chan fqdnResolutionResult, getIPInfo func(net.Interface) ([]string, map[string]string), getFQDN func(string, *net.Interface) string) (string, error) {
	ips, _ := getIPInfo(iface)

	// Local FQDN resolution (fast)
	localFQDN := getFQDN(hostname, &iface)
	log.Debug().Interface("fqdn", localFQDN).Msg("Local FQDN resolution result")

	// Global FQDN resolution (wait if needed)
	var globalFQDN string
	select {
	case res := <-fqdnChan:
		globalFQDN = res.fqdn
	default:
		s := tap.NewSpinner(tap.SpinnerOptions{})
		s.Start("Resolving network information...")
		res := <-fqdnChan
		s.Stop("Network information resolved", 0)
		globalFQDN = res.fqdn
	}
	log.Debug().Interface("fqdn", globalFQDN).Msg("Global FQDN resolution result")

	hostAddress := tap.Select(ctx, tap.SelectOptions[string]{
		Message: "Host Address (Used for communication with the daemon)",
		Options: BuildHostOptions(hostname, globalFQDN, localFQDN, ips),
	})
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	log.Debug().Interface("address", hostAddress).Msg("Selected host address")
	return hostAddress, nil
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

// AssembleConfig is a pure function that populates the Config and State structs.
func AssembleConfig(hostAddress string, mac string, wolAddress string, agentPort int, reportsBoot bool, grubWait int, grubPath string) *config.Config {
	cfg := &config.Config{
		Host: config.HostConfig{
			Address: hostAddress,
			MAC:     mac,
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
