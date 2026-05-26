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
	configPath := config.DefaultConfigPath()
	// Check if already configured
	isReinstall := false
	currentPort := 0
	if cfg, err := config.LoadConfig(configPath); err == nil {
		isReinstall = true
		currentPort = cfg.Daemon.Port
	}

	// 0. Preflight check
	if err := stepConfirmOverwrite(isReinstall, isDryRun); err != nil {
		return nil, err
	}

	// 1. Detect GRUB and prompt for boot reporting
	g := grub.NewGrub()
	grubConfigPath, _ := g.DiscoverConfigPath()
	var reportsBoot bool
	if grubConfigPath != "" {
		var err error
		reportsBoot, err = stepPromptReportBootOptions(grubConfigPath)
		if err != nil {
			return nil, err
		}
	}

	// 2. Prepare Network Interface and WOL options
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list network interfaces: %w", err)
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
	portStr := strconv.Itoa(config.DefaultAgentPort)
	if currentPort > 0 {
		portStr = strconv.Itoa(currentPort)
	}
	wolAddr := config.DefaultWolBroadcastAddress
	waitStr := strconv.Itoa(config.DefaultGrubWaitSeconds)

	fields := []huh.Field{
		huh.NewSelect[int]().
			Title("Select network interface to use").
			Options(ifaceOptions...).
			Value(&selectedIfaceIdx),
		huh.NewInput().
			Title("Daemon Port").
			Value(&portStr).
			Placeholder(strconv.Itoa(config.DefaultAgentPort)).
			Validate(func(s string) error {
				if s == "" {
					return nil
				}
				p, err := strconv.Atoi(s)
				if err != nil {
					return fmt.Errorf("invalid port: %w", err)
				}
				if p < 1 || p > 65535 {
					return fmt.Errorf("port must be between 1 and 65535")
				}
				return nil
			}),
		huh.NewSelect[string]().
			Title("WOL Broadcast Address").
			Description("Choose subnet broadcast if you have cross-VLAN setups").
			Options(wolOptions...).
			Value(&wolAddr),
	}

	if reportsBoot {
		fields = append(fields, huh.NewInput().
			Title("GRUB Network Wait").
			Description("seconds to wait for network during boot").
			Value(&waitStr).
			Placeholder(strconv.Itoa(config.DefaultGrubWaitSeconds)).
			Validate(func(s string) error {
				if s == "" {
					return nil
				}
				return config.ValidateGrubWaitTime(s)
			}))
	}

	err = huh.NewForm(huh.NewGroup(fields...)).Run()
	if err != nil {
		return nil, err
	}

	selectedIface := filtered[selectedIfaceIdx]
	port, _ := strconv.Atoi(portStr)
	grubWaitTime, _ := strconv.Atoi(waitStr)

	cfg := AssembleConfig(selectedIface.Name, port, wolAddr, reportsBoot, grubWaitTime, grubConfigPath)
	return cfg, nil
}

func stepConfirmOverwrite(isReinstall, isDryRun bool) error {
	if isReinstall && !isDryRun {
		var overwrite bool
		err := huh.NewConfirm().
			Title("GrubStation is already configured. Do you want to re-run setup and overwrite the existing configuration?").
			Value(&overwrite).
			Run()
		if err != nil {
			return err
		}
		if !overwrite {
			return ErrAborted
		}
	}
	return nil
}

func stepPromptReportBootOptions(grubPath string) (bool, error) {
	var reportsBoot bool
	err := huh.NewConfirm().
		Title(fmt.Sprintf("Enable remote boot selection? (Detected GRUB at %s)", grubPath)).
		Value(&reportsBoot).
		Run()
	if err != nil {
		return false, err
	}
	log.Debug().Interface("reportsBoot", reportsBoot).Msg("User selected boot reporting preference")
	return reportsBoot, nil
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
