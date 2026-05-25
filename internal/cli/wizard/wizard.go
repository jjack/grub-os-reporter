package wizard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/homeassistant"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/yarlson/tap"
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
	haChan := startHADiscovery(ctx, discoverHA)
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

	// 7. Home Assistant URL
	haURL, grubURL, err := stepSelectHomeAssistantURL(ctx, haChan)
	if err != nil {
		return nil, err
	}

	// 8. HA Webhook ID
	haWebhook, err := stepGetWebhookID(ctx)
	if err != nil {
		return nil, err
	}

	return AssembleConfig(hostAddress, selectedIface.HardwareAddr.String(), wolBroadcastAddress, haURL, haWebhook, agentPort, reportsBoot, grubWaitTime, finalGrubConfigPath, grubURL), nil
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

type haDiscoveryResult struct {
	instances []homeassistant.ServiceInstance
	err       error
}

func startHADiscovery(ctx context.Context, discoverHA func(context.Context) ([]homeassistant.ServiceInstance, error)) <-chan haDiscoveryResult {
	haChan := make(chan haDiscoveryResult, 1)
	go func() {
		log.Debug().Msg("Starting background Home Assistant discovery")
		instances, err := discoverHA(ctx)
		if err != nil {
			log.Debug().Err(err).Msg("Background HA discovery failed")
		} else {
			log.Debug().Interface("count", len(instances)).Msg("Background HA discovery complete")
		}
		haChan <- haDiscoveryResult{instances, err}
	}()
	return haChan
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

func stepSelectHomeAssistantURL(ctx context.Context, haChan <-chan haDiscoveryResult) (string, string, error) {
	var discovered []homeassistant.ServiceInstance
	select {
	case res := <-haChan:
		discovered = res.instances
	default:
		s := tap.NewSpinner(tap.SpinnerOptions{})
		s.Start("Discovering Home Assistant...")
		res := <-haChan
		s.Stop("Home Assistant discovery complete", 0)
		discovered = res.instances
	}

	var haURL string
	var grubURL string

	totalURLs := 0
	for _, inst := range discovered {
		totalURLs += len(inst.URLs)
	}

	if totalURLs == 1 {
		haURL = discovered[0].URLs[0]
		log.Debug().Interface("url", haURL).Msg("Single Home Assistant instance discovered, using it")
	} else if len(discovered) > 0 {
		instIdx, err := selectHAInstance(ctx, discovered)
		if err != nil {
			return "", "", err
		}

		if instIdx != -1 {
			selectedInst := discovered[instIdx]
			log.Debug().Interface("name", selectedInst.Name).Msg("Selected Home Assistant instance")

			var err error
			haURL, err = selectHAURLForAgent(ctx, selectedInst)
			if err != nil {
				return "", "", err
			}

			if strings.HasPrefix(haURL, "https://") {
				grubURL, _ = selectHAURLForGRUB(ctx, selectedInst)
				if ctx.Err() != nil {
					return "", "", ctx.Err()
				}
			}
		}
	}

	if haURL == "" {
		var err error
		haURL, err = enterHAURLManually(ctx)
		if err != nil {
			return "", "", err
		}
	}

	return haURL, grubURL, nil
}

func selectHAInstance(ctx context.Context, discovered []homeassistant.ServiceInstance) (int, error) {
	var instOpts []tap.SelectOption[int]
	for i, inst := range discovered {
		label := fmt.Sprintf("%s (%s)", inst.Name, strings.Join(inst.URLs, ", "))
		instOpts = append(instOpts, tap.SelectOption[int]{Value: i, Label: label})
	}
	instOpts = append(instOpts, tap.SelectOption[int]{Value: -1, Label: "Other (Enter manually)"})

	idx := tap.Select(ctx, tap.SelectOptions[int]{
		Message: "Home Assistant instances discovered. Please select one:",
		Options: instOpts,
	})
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}
	return idx, nil
}

func selectHAURLForAgent(ctx context.Context, inst homeassistant.ServiceInstance) (string, error) {
	var agentOpts []tap.SelectOption[string]
	for _, u := range inst.URLs {
		label := u
		if strings.HasPrefix(u, "https://") {
			label += " (HTTPS is Preferred)"
		}
		agentOpts = append(agentOpts, tap.SelectOption[string]{Value: u, Label: label})
	}

	url := tap.Select(ctx, tap.SelectOptions[string]{
		Message: fmt.Sprintf("Select URL for the Agent (%s):", inst.Name),
		Options: agentOpts,
	})
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	log.Debug().Interface("url", url).Msg("Selected Home Assistant URL")
	return url, nil
}

func selectHAURLForGRUB(ctx context.Context, inst homeassistant.ServiceInstance) (string, error) {
	var grubOpts []tap.SelectOption[string]
	for _, u := range inst.URLs {
		if !strings.HasPrefix(u, "https://") {
			grubOpts = append(grubOpts, tap.SelectOption[string]{Value: u, Label: u})
		}
	}

	if len(grubOpts) > 0 {
		url := tap.Select(ctx, tap.SelectOptions[string]{
			Message: "Select HTTP URL for GRUB (HTTPS not readily supported in GRUB):",
			Options: grubOpts,
		})
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		log.Debug().Interface("url", url).Msg("Selected GRUB HTTP URL")
		return url, nil
	}
	return "", nil
}

func enterHAURLManually(ctx context.Context) (string, error) {
	haURL := tap.Text(ctx, tap.TextOptions{
		Message: "Home Assistant URL",
		Validate: func(s string) error {
			skipCheck := os.Getenv("GRUBSTATION_SKIP_HA_URL_CHECK") == "true"
			return ValidateHAURL(ctx, s, skipCheck, CheckHAConnection)
		},
	})
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	log.Debug().Interface("url", haURL).Msg("Manually entered Home Assistant URL")
	return haURL, nil
}

func stepGetWebhookID(ctx context.Context) (string, error) {
	haWebhook := tap.Password(ctx, tap.PasswordOptions{
		Message: "Home Assistant Webhook ID (generated by the integration)",
		Validate: func(s string) error {
			return config.ValidateWebhookID(s)
		},
	})
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	log.Debug().Msg("Home Assistant Webhook ID provided and validated")
	return haWebhook, nil
}

// AssembleConfig is a pure function that populates the Config struct.
func AssembleConfig(hostAddress, mac, wolAddress, haURL, haWebhook string, agentPort int, reportsBoot bool, grubWait int, grubPath, grubURL string) *config.Config {
	return &config.Config{
		Host: config.HostConfig{
			Address:    hostAddress,
			MACAddress: mac,
		},
		WakeOnLan: &config.WakeOnLanConfig{
			Address: wolAddress,
		},
		HomeAssistant: config.HomeAssistantConfig{
			URL:       haURL,
			WebhookID: haWebhook,
		},
		Daemon: config.DaemonConfig{
			Port:              agentPort,
			ReportBootOptions: reportsBoot,
		},
		Grub: &config.GrubConfig{
			WaitTimeSeconds: grubWait,
			ConfigPath:      grubPath,
			URL:             grubURL,
		},
	}
}

func PrintConfigSummary(cmd *cobra.Command, cfg *config.Config, cfgPath string) {
	exporter := &config.Exporter{
		Config:     *cfg,
		Mask:       true,
		Exhaustive: false,
	}
	out, err := exporter.ToYAML()
	if err != nil {
		tap.Message(fmt.Sprintf("Error generating summary: %v", err))
		return
	}

	tap.Message(fmt.Sprintf("Configuration saved to %s", cfgPath))
	out = fmt.Sprintf("\n---\n%s", out)
	tap.Box(out, " Configuration Preview ", tap.BoxOptions{
		ContentPadding: 2,
	})
}
