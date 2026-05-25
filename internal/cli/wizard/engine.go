package wizard

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jjack/grubstation/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/yarlson/tap"
)

// Mode constants
const (
	ModeDaemonBoth     = "Daemon (Remote shutdown + Report boot options)"
	ModeDaemonShutdown = "Daemon (Remote shutdown only)"
	ModeHookOnly       = "Shutdown hook (Report boot options only)"
)

// GetModeOptions returns the available installation modes based on whether a GRUB config was found.
func GetModeOptions(grubConfigPath string) []tap.SelectOption[string] {
	if grubConfigPath != "" {
		return []tap.SelectOption[string]{
			{Value: ModeDaemonBoth, Label: ModeDaemonBoth},
			{Value: ModeDaemonShutdown, Label: ModeDaemonShutdown},
			{Value: ModeHookOnly, Label: ModeHookOnly},
		}
	}
	return []tap.SelectOption[string]{
		{Value: ModeDaemonShutdown, Label: ModeDaemonShutdown},
	}
}

// GetModeFlags converts a selected mode string into boolean flags.
func GetModeFlags(mode string) (reportsBoot, runsDaemon bool) {
	reportsBoot = mode == ModeDaemonBoth || mode == ModeHookOnly
	runsDaemon = mode == ModeDaemonBoth || mode == ModeDaemonShutdown
	return
}

// BuildIfaceOptions builds the selection options for network interfaces.
func BuildIfaceOptions(interfaces []net.Interface, ipProvider func(net.Interface) ([]string, map[string]string)) []tap.SelectOption[int] {
	var opts []tap.SelectOption[int]
	for i, inf := range interfaces {
		ips, _ := ipProvider(inf)
		desc := fmt.Sprintf("(%s) [%s]", inf.HardwareAddr.String(), strings.Join(ips, ", "))
		opts = append(opts, tap.SelectOption[int]{
			Value: i,
			Label: inf.Name,
			Hint:  desc,
		})
	}
	return opts
}

// BuildHostOptions builds the selection options for the host address.
func BuildHostOptions(hostname, globalFQDN, localFQDN string, ips []string) []tap.SelectOption[string] {
	var opts []tap.SelectOption[string]
	seen := make(map[string]bool)

	addOption := func(val, hint string) {
		if val != "" && !seen[val] {
			seen[val] = true
			opts = append(opts, tap.SelectOption[string]{Value: val, Label: val, Hint: hint})
		}
	}

	if localFQDN != "" {
		addOption(localFQDN, "Local FQDN (from selected adapter)")
	}
	if globalFQDN != "" {
		addOption(globalFQDN, "Global FQDN (via DNS)")
	}

	addOption(hostname, "Hostname")

	for _, ip := range ips {
		addOption(ip, "IP Address [Ensure This is Static!]")
	}
	return opts
}

// BuildWolOptions builds the selection options for the WOL broadcast address.
func BuildWolOptions(ips []string, ipBroadcasts map[string]string) []tap.SelectOption[string] {
	opts := []tap.SelectOption[string]{
		{Value: config.DefaultWolBroadcastAddress, Label: fmt.Sprintf("%s (Default)", config.DefaultWolBroadcastAddress)},
	}

	seenBroadcasts := make(map[string]bool)
	for _, ip := range ips {
		bc, ok := ipBroadcasts[ip]
		if !ok {
			continue
		}

		if !seenBroadcasts[bc] {
			seenBroadcasts[bc] = true
			opts = append(opts, tap.SelectOption[string]{
				Value: bc,
				Label: fmt.Sprintf("%s (Subnet broadcast for %s)", bc, ip),
			})
		}
	}
	return opts
}

// ValidatePort checks if a port is valid and available.
func ValidatePort(s string, isReinstall bool, currentPort int, portChecker func(int) error) error {
	if err := config.ValidatePort(s); err != nil {
		return err
	}
	port, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	if isReinstall && port == currentPort {
		return nil
	}

	return portChecker(port)
}

// CheckPortAvailability is the default implementation of the port checker.
func CheckPortAvailability(port int) error {
	log.Debug().Interface("port", port).Msg("Checking port availability")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Debug().Interface("port", port).Err(err).Msg("Port availability check failed")
		return fmt.Errorf("port %d is in use or unavailable: %v", port, err)
	}
	_ = listener.Close()
	log.Debug().Interface("port", port).Msg("Port is available")
	return nil
}

// ValidateHAURL checks if a Home Assistant URL is valid and reachable.
func ValidateHAURL(ctx context.Context, s string, skipCheck bool, urlChecker func(context.Context, string) error) error {
	if err := config.ValidateURL(s); err != nil {
		return err
	}

	if skipCheck || urlChecker == nil {
		log.Debug().Msg("Skipping Home Assistant URL connection check as requested")
		return nil
	}

	return urlChecker(ctx, s)
}

// CheckHAConnection is the default implementation of the HA URL connection checker.
func CheckHAConnection(ctx context.Context, url string) error {
	log.Debug().Interface("url", url).Msg("Checking Home Assistant connection")
	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Debug().Interface("url", url).Err(err).Msg("Failed to create request for HA connection check")
		return fmt.Errorf("invalid request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Debug().Interface("url", url).Err(err).Msg("Home Assistant connection check failed")
		return fmt.Errorf("could not connect to HA URL: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	log.Debug().Interface("url", url).Interface("status", resp.StatusCode).Msg("Home Assistant connection check successful")
	return nil
}
