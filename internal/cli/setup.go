package cli

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jjack/grubstation/internal/cli/wizard"
	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/grub"
	"github.com/jjack/grubstation/internal/homeassistant"
	"github.com/jjack/grubstation/internal/host"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/yarlson/tap"
)

func NewSetupCmd() *cobra.Command {
	var applyOnly bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Run the automated setup wizard to configure and install the agent",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if applyOnly {
				cfg, cfgFile, err := GetConfig(cmd)
				if err != nil {
					return err
				}

				state, _ := config.LoadState(cfgFile)
				if !state.Paired {
					return fmt.Errorf("system is not paired. Run the full setup or pair via API first")
				}

				log.Info().Msg("Applying GRUB configuration...")
				g := grub.NewGrub()
				if err := g.Setup(grub.SetupOptions{
					TargetMAC:       cfg.Host.MAC,
					TargetURL:       state.HAGrubURL,
					AuthToken:       state.WebhookID,
					WaitTimeSeconds: cfg.Grub.NetworkWaitTime,
				}); err != nil {
					return err
				}

				return nil
			}
			return nil // Override root config loading for wizard
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			dump := setupDebugLogging()
			defer func() { dump(err) }()

			if runtime.GOOS == "windows" {
				defer func() {
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
				_, cfgPath, _ := GetConfig(cmd)
				return performInstall(cmd, cfgPath, "")
			}

			cfg, err := wizard.RunGenerateSurvey(dryRun, host.New().GetIPInfo)
			if err != nil {
				if errors.Is(err, wizard.ErrAborted) {
					tap.Message("Setup aborted.")
					tap.Outro("Goodbye!")
					return nil
				}
				return err
			}

			// Perform technical "state data" discovery after the wizard
			if err := populateTechnicalConfig(cfg); err != nil {
				return err
			}

			// Determine final config path
			_, cfgPath, _ := GetConfig(cmd)

			if dryRun {
				return doDryRun(cfg, cfgPath)
			}

			return doInstallation(cmd, cfg, cfgPath)
		},
	}

	cmd.Flags().BoolVar(&applyOnly, "apply", false, "Skip survey and install service based on current config")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview configuration without saving or installing")

	return cmd
}

func populateTechnicalConfig(cfg *config.Config) error {
	iface, err := net.InterfaceByName(cfg.Host.Interface)
	if err != nil {
		return fmt.Errorf("failed to get interface %s: %w", cfg.Host.Interface, err)
	}

	cfg.Host.MAC = iface.HardwareAddr.String()

	// Ensure defaults for optional fields if not set by wizard (though it usually is)
	if cfg.WakeOnLan.Port == 0 {
		cfg.WakeOnLan.Port = config.DefaultWolBroadcastPort
	}
	if cfg.WakeOnLan.Address == "" {
		cfg.WakeOnLan.Address = config.DefaultWolBroadcastAddress
	}

	return nil
}

func doDryRun(cfg *config.Config, cfgPath string) error {
	wizard.PrintConfigSummary(os.Stdout, cfg, cfgPath)

	mgr, err := GetManager()
	if err != nil {
		return err
	}

	if svcPreview, err := mgr.Preview(cfgPath); err == nil {
		tap.Box(svcPreview, fmt.Sprintf(" %s Service Preview ", mgr.Name()), tap.BoxOptions{
			ContentPadding: 2,
		})
	}

	if cfg.Daemon.ReportBootOptions {
		waitTime := config.DefaultGrubWaitSeconds
		if cfg.Grub.NetworkWaitTime != 0 {
			waitTime = cfg.Grub.NetworkWaitTime
		}

		g := grub.NewGrub()
		grubPreview, err := g.GenerateScript(grub.SetupOptions{
			TargetMAC:       cfg.Host.MAC,
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

func doInstallation(cmd *cobra.Command, cfg *config.Config, cfgPath string) error {
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := config.Save(cfg, cfgPath); err != nil {
		return err
	}

	// Initialize empty state
	pairState := &config.State{Paired: false}
	if err := pairState.Save(cfgPath); err != nil {
		return err
	}

	tap.Outro("Configuration setup complete.", tap.MessageOptions{
		Hint: fmt.Sprintf("saved to: %s", cfgPath),
	})

	tap.Intro("Proceeding with installation...")

	if err := performInstall(cmd, cfgPath, ""); err != nil {
		return err
	}

	tap.Outro("Setup complete!")
	return nil
}

func performInstall(cmd *cobra.Command, cfgFile string, token string) error {
	log.Debug().Interface("config", cfgFile).Msg("Starting installation process")
	cfg, _, err := GetConfig(cmd)
	if err != nil {
		return err
	}

	mgr, err := GetManager()
	if err != nil {
		return err
	}

	if err := mgr.CheckPermissions(); err != nil {
		return err
	}

	absConfig, err := filepath.Abs(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	if cfg.Daemon.ReportBootOptions {
		waitTime := config.DefaultGrubWaitSeconds
		if cfg.Grub.NetworkWaitTime != 0 {
			waitTime = cfg.Grub.NetworkWaitTime
		}

		// Load state for GRUB setup details
		state, _ := config.LoadState(cfgFile)
		targetURL := state.HAGrubURL
		if cfg.Grub.URL != "" {
			targetURL = cfg.Grub.URL
		}

		opts := grub.SetupOptions{
			TargetMAC:       cfg.Host.MAC,
			TargetURL:       targetURL,
			AuthToken:       state.WebhookID,
			WaitTimeSeconds: waitTime,
		}

		g := grub.NewGrub()
		warning := g.SetupWarning()
		tap.Message("Installing into grub...", tap.MessageOptions{
			Hint: warning,
		})

		if err := g.Setup(opts); err != nil {
			return fmt.Errorf("failed to install grub: %w", err)
		}

		if state.HADaemonURL != "" && state.WebhookID != "" {
			tap.Message("Pushing initial boot options to Home Assistant...")
			haClient := homeassistant.NewClient(state.HADaemonURL, state.WebhookID, nil)

			g := grub.NewGrub()
			options, err := g.GetBootOptions()
			if err != nil {
				return err
			}

			if err := haClient.UpdateBootOptions(cfg, state, options); err != nil {
				return err
			}
			tap.Message("Successfully pushed initial state to Home Assistant.")
		}
	}

	tap.Message(fmt.Sprintf("Installing into service manager: %s", mgr.Name()))
	if err := mgr.Configure(cfg); err != nil {
		return fmt.Errorf("failed to configure service: %w", err)
	}

	if err := mgr.Install(absConfig); err != nil {
		return fmt.Errorf("failed to install manager: %w", err)
	}

	tap.Message("Starting service...")
	if err := mgr.Start(); err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	tap.Message("Installation completed successfully.")
	return nil
}

func IsInstalled() (bool, error) {
	mgr, err := GetManager()
	if err != nil {
		return false, err
	}
	return mgr.IsInstalled()
}
