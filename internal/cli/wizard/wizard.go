package wizard

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"

	"charm.land/huh/v2"
	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/grub"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

var (
	RunGenerateSurvey func(bool, func(net.Interface) ([]string, map[string]string)) (*config.Config, error) = generateConfigInteractive

	ErrAborted = errors.New("setup aborted")
)

func generateConfigInteractive(isDryRun bool, getIPInfo func(net.Interface) ([]string, map[string]string)) (*config.Config, error) {
	fmt.Print("\033[H\033[2J")

	// 0. Preflight check to see  if already configured
	cfgPath := config.DefaultConfigPath()
	isReinstall := false
	currentPort := "0"
	if !isDryRun {
		if c, err := config.LoadConfig(cfgPath); err == nil {
			err := huh.NewConfirm().
				Title("GrubStation is already configured. Do you want to re-run setup and overwrite the existing configuration?").
				Value(&isReinstall).
				Run()
			if err != nil {
				return nil, err
			}
			if !isReinstall {
				return nil, ErrAborted
			}
			currentPort = strconv.Itoa(c.Daemon.Port)

			fmt.Print("\033[H\033[2J")
		}
	}

	cfg := &config.Config{}

	// 1. Detect GRUB and prompt for boot reporting
	g := grub.NewGrub()
	grubConfigPath, _ := g.DiscoverConfigPath()
	if grubConfigPath == "" {
		cfg.Daemon.ReportBootOptions = false
	} else {
		cfg.Grub.Path = grubConfigPath
		err := huh.NewConfirm().
			Title(fmt.Sprintf("Report GRUB boot options? (Detected GRUB at %s)", grubConfigPath)).
			Value(&cfg.Daemon.ReportBootOptions).
			Run()
		if err != nil {
			return nil, err
		}
		log.Debug().Bool("reportsBoot", cfg.Daemon.ReportBootOptions).Msg("User selected boot reporting preference")
		fmt.Print("\033[H\033[2J")
	}

	// 2. Prepare Network Interface and WOL options
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var filtered []net.Interface
	var wolOptions []huh.Option[string]
	wolOptions = append(wolOptions, huh.NewOption(fmt.Sprintf("%s (Default)", config.DefaultWolBroadcastAddress), config.DefaultWolBroadcastAddress))

	for _, inf := range interfaces {
		if inf.Flags&net.FlagUp != 0 && inf.Flags&net.FlagLoopback == 0 && len(inf.HardwareAddr) > 0 {
			filtered = append(filtered, inf)

			// Collect broadcasts for this interface
			ips, broadcasts := getIPInfo(inf)
			for _, ip := range ips {
				if bc, ok := broadcasts[ip]; ok {
					label := fmt.Sprintf("%s (Subnet for %s on %s)", bc, ip, inf.Name)
					wolOptions = append(wolOptions, huh.NewOption(label, bc))
				}
			}
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("no suitable network interfaces found")
	}

	ifaceOptions := BuildIfaceOptions(filtered, getIPInfo)

	// 3. Main Configuration Group
	var selectedIfaceIdx int
	waitStr := strconv.Itoa(config.DefaultGrubWaitSeconds)

	fields := []huh.Field{
		huh.NewSelect[int]().
			Title("Select network interface to use").
			Options(ifaceOptions...).
			Value(&selectedIfaceIdx),
		huh.NewInput().
			Title("Daemon Port").
			Value(&currentPort).
			Placeholder(strconv.Itoa(config.DefaultAgentPort)).
			Validate(func(s string) error {
				if s == "" {
					return nil
				}
				return config.ValidatePort(s)
			}),
		huh.NewSelect[string]().
			Title("WOL Broadcast Address").
			Description("Choose subnet broadcast if you have cross-VLAN setups").
			Options(wolOptions...).
			Value(&cfg.WakeOnLan.Address),
	}

	if cfg.Daemon.ReportBootOptions {
		fields = append(fields, huh.NewInput().
			Title("GRUB Network Wait").
			Description("seconds to wait for network during boot").
			Value(&waitStr).
			Placeholder(strconv.Itoa(config.DefaultGrubWaitSeconds)).
			Validate(config.ValidateGrubWaitTime))
	}

	err = huh.NewForm(huh.NewGroup(fields...)).Run()
	if err != nil {
		return nil, err
	}

	cfg.Host.Interface = filtered[selectedIfaceIdx].Name
	cfg.Daemon.Port, _ = strconv.Atoi(currentPort)
	cfg.Grub.NetworkWaitTime, _ = strconv.Atoi(waitStr)

	return cfg, nil
}

func PrintConfigSummary(w io.Writer, cfg *config.Config, cfgPath string) {
	out, err := yaml.Marshal(cfg.Minimal())
	if err != nil {
		_, _ = fmt.Fprintf(w, "Error generating summary: %v\n", err)
		return
	}

	_, _ = fmt.Fprintf(w, "\nConfiguration saved to %s\n", cfgPath)
	_, _ = fmt.Fprintln(w, "---")
	_, _ = fmt.Fprintln(w, string(out))
	_, _ = fmt.Fprintln(w, "---")
}
