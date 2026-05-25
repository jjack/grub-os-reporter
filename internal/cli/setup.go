package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jjack/grubstation/internal/cli/wizard"
	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/grub"
	"github.com/jjack/grubstation/internal/homeassistant"
	"github.com/jjack/grubstation/internal/servicemanager"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yarlson/tap"
)

var osMkdirAll = os.MkdirAll

var ErrElevated = errors.New("elevated")

func NewSetupCmd(deps *CommandDeps) *cobra.Command {
	var applyOnly bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Run the automated setup wizard to configure and install the agent",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if applyOnly {
				// Use a temporary CLI instance to load config
				tempCLI := &CLI{}
				cfgFile, _ := cmd.Flags().GetString("config")
				if err := tempCLI.LoadConfig(cmd, cfgFile); err != nil {
					return err
				}
				deps.Config = tempCLI.Config
				deps.ConfigFile = cfgFile
				return nil
			}
			return nil // Override root config loading for wizard
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			dump := setupDebugLogging()
			defer func() { dump(err) }()

			if runtime.GOOS == "windows" {
				defer func() {
					if err == ErrElevated {
						return
					}

					if err != nil {
						tap.Outro(fmt.Sprintf("Error: %v", err))
					}

					// Always wait on Windows when running the setup wizard so the user can see the output.
					// We tell the user they can close the window manually because stdin can be unreliable in some MSI-launched environments.
					fmt.Print("\nSetup finished. You can now close this window.")
					s := bufio.NewScanner(os.Stdin)
					s.Scan()
				}()
			}

			if applyOnly {
				cfgPath, _ := cmd.Flags().GetString("config")
				return performInstall(cmd, deps, cfgPath, "")
			}

			mgr, err := ensureSupport(cmd.Context(), deps)
			if err != nil {
				return err
			}

			if !dryRun {
				if err := mgr.CheckPermissions(cmd.Context()); err != nil {
					return err
				}
			}

			cfgPath, err := cmd.Flags().GetString("config")
			if err != nil || cfgPath == "" {
				cfgPath = config.DefaultConfigPath()
			}

			var currentPort int
			if _, err := os.Stat(cfgPath); err == nil {
				v := viper.New()
				v.SetConfigFile(cfgPath)
				if err := v.ReadInConfig(); err == nil {
					currentPort = v.GetInt("daemon.port")
				}
			}

			cfg, err := doWizard(cmd.Context(), deps, cfgPath, currentPort, dryRun)
			if err != nil {
				return err
			}
			if cfg == nil {
				return nil // Aborted
			}

			if dryRun {
				return doDryRun(cmd, deps, cfg, cfgPath, mgr)
			}

			return doInstallation(cmd, deps, cfg, cfgPath)
		},
	}

	cmd.Flags().BoolVar(&applyOnly, "apply", false, "Skip survey and install service based on current config")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview configuration without saving or installing")

	return cmd
}

func doWizard(ctx context.Context, deps *CommandDeps, cfgPath string, currentPort int, dryRun bool) (*config.Config, error) {
	// Clear the terminal screen before starting the interactive wizard
	fmt.Print("\033[H\033[2J")
	tap.Intro("GrubStation Setup")

	isConfigured := false
	if _, err := os.Stat(cfgPath); err == nil {
		isConfigured = true
	}

	// Perform initial discovery
	hostname, _ := deps.Host.DetectHostname()
	interfaces, _ := deps.Host.GetWOLInterfaces()
	grubConfigPath, _ := deps.Grub.DiscoverConfigPath(ctx)

	state := wizard.SystemState{
		Hostname:       hostname,
		Interfaces:     interfaces,
		GrubConfigPath: grubConfigPath,
		IsReinstall:    isConfigured,
		CurrentPort:    currentPort,
	}

	cfg, err := wizard.RunGenerateSurvey(ctx, state, dryRun, deps.Host.GetIPInfo, deps.Host.GetFQDN, deps.DiscoverHA)
	if err != nil {
		if errors.Is(err, wizard.ErrAborted) {
			tap.Message("Setup aborted.")
			tap.Outro("Goodbye!")
			return nil, nil
		}
		return nil, err
	}
	return cfg, nil
}

func doDryRun(cmd *cobra.Command, deps *CommandDeps, cfg *config.Config, cfgPath string, mgr servicemanager.Manager) error {
	wizard.PrintConfigSummary(cmd, cfg, cfgPath)

	if svcPreview, err := mgr.Preview(cmd.Context(), cfgPath); err == nil {
		tap.Box(svcPreview, fmt.Sprintf(" %s Service Preview ", mgr.Name()), tap.BoxOptions{
			ContentPadding: 2,
		})
	}

	if cfg.Daemon.ReportBootOptions {
		waitTime := config.DefaultGrubWaitSeconds
		targetURL := cfg.HomeAssistant.URL
		if cfg.Grub != nil {
			waitTime = cfg.Grub.WaitTimeSeconds
			if cfg.Grub.URL != "" {
				targetURL = cfg.Grub.URL
			}
		}
		grubPreview, err := deps.Grub.GenerateScript(grub.SetupOptions{
			TargetMAC:       cfg.Host.MACAddress,
			TargetURL:       targetURL,
			AuthToken:       cfg.HomeAssistant.WebhookID,
			WaitTimeSeconds: waitTime,
		})
		if err == nil {
			tap.Box(grubPreview, " GRUB Script Preview (/etc/grub.d/99_grubstation) ", tap.BoxOptions{
				ContentPadding: 2,
			})
		}
	}

	tap.Message("Dry run completed. Configuration shown above was not saved.")
	tap.Outro("Dry run finished")
	return nil
}

func doInstallation(cmd *cobra.Command, deps *CommandDeps, cfg *config.Config, cfgPath string) error {
	if err := osMkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := config.Save(cfg, cfgPath); err != nil {
		return err
	}

	tap.Outro("Configuration setup complete.", tap.MessageOptions{
		Hint: fmt.Sprintf("saved to: %s", cfgPath),
	})

	tap.Intro("Proceeding with installation...")

	// We update the deps config with our freshly generated config so the installer can use it
	*deps.Config = *cfg

	if err := performInstall(cmd, deps, cfgPath, ""); err != nil {
		if err == ErrElevated {
			return nil
		}
		return err
	}

	tap.Outro("Setup complete!")
	return nil
}

func performInstall(cmd *cobra.Command, deps *CommandDeps, cfgFile string, token string) error {
	log.Debug().Interface("config", cfgFile).Msg("Starting installation process")
	mgr, err := deps.Manager(cmd.Context())
	if err != nil {
		return err
	}

	if err := mgr.CheckPermissions(cmd.Context()); err != nil {
		return err
	}

	absConfig, err := filepath.Abs(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	if deps.Config.Daemon.ReportBootOptions {
		waitTime := config.DefaultGrubWaitSeconds
		targetURL := deps.Config.HomeAssistant.URL
		if deps.Config.Grub != nil {
			waitTime = deps.Config.Grub.WaitTimeSeconds
			if deps.Config.Grub.URL != "" {
				targetURL = deps.Config.Grub.URL
			}
		}

		opts := grub.SetupOptions{
			TargetMAC:       deps.Config.Host.MACAddress,
			TargetURL:       targetURL,
			AuthToken:       deps.Config.HomeAssistant.WebhookID,
			WaitTimeSeconds: waitTime,
		}

		warning := deps.Grub.SetupWarning()
		tap.Message("Installing into grub...", tap.MessageOptions{
			Hint: warning,
		})

		if err := deps.Grub.Setup(cmd.Context(), opts); err != nil {
			return fmt.Errorf("failed to install grub: %w", err)
		}

		if deps.Config.HomeAssistant.URL != "" && deps.Config.HomeAssistant.WebhookID != "" {
			tap.Message("Pushing initial boot options to Home Assistant...")
			haClient := homeassistant.NewClient(deps.Config.HomeAssistant.URL, deps.Config.HomeAssistant.WebhookID, nil)

			if token != "" {
				if err := haClient.RegisterAgent(cmd.Context(), deps.Config.Host.MACAddress, deps.Config.Host.Address, token, deps.Config.Daemon.Port); err != nil {
					return err
				}
			}

			options, err := deps.Grub.GetBootOptions(cmd.Context())
			if err != nil {
				return err
			}

			var wolAddr string
			var wolPort int
			if deps.Config.WakeOnLan != nil {
				wolAddr = deps.Config.WakeOnLan.Address
				wolPort = deps.Config.WakeOnLan.Port
			}

			if err := haClient.UpdateBootOptions(cmd.Context(), deps.Config.Host.MACAddress, deps.Config.Host.Address, options, wolAddr, wolPort); err != nil {
				return err
			}
			tap.Message("Successfully pushed initial state to Home Assistant.")
		}
	}

	tap.Message(fmt.Sprintf("Installing into service manager: %s", mgr.Name()))
	if err := mgr.Configure(cmd.Context(), deps.Config); err != nil {
		return fmt.Errorf("failed to configure service: %w", err)
	}

	if err := mgr.Install(cmd.Context(), absConfig); err != nil {
		return fmt.Errorf("failed to install manager: %w", err)
	}

	tap.Message("Starting service...")
	if err := mgr.Start(cmd.Context()); err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	tap.Message("Installation completed successfully.")
	return nil
}

func ensureSupport(ctx context.Context, deps *CommandDeps) (servicemanager.Manager, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	mgr, err := deps.Registry.Detect(ctx)
	if err != nil {
		if errors.Is(err, servicemanager.ErrNotSupported) {
			supported := strings.Join(deps.Registry.SupportedServices(), ", ")
			return nil, fmt.Errorf("no supported service manager detected. Please ensure you have one of the following installed: %s", supported)
		}
		return nil, err
	}
	return mgr, nil
}

func IsInstalled(ctx context.Context, deps *CommandDeps) (bool, error) {
	mgr, err := ensureSupport(ctx, deps)
	if err != nil {
		return false, err
	}
	return mgr.IsInstalled(ctx)
}
