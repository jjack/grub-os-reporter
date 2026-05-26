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

var ErrElevated = errors.New("elevated")

func NewSetupCmd() *cobra.Command {
	var applyOnly bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Run the automated setup wizard to configure and install the agent",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if applyOnly {
				env, err := GetEnv(cmd)
				if err != nil {
					return err
				}

				if !env.State.Paired {
					return fmt.Errorf("system is not paired. Run the full setup or pair via API first")
				}

				log.Info().Msg("Applying GRUB configuration...")
				if err := env.Grub.Setup(grub.SetupOptions{
					TargetMAC:       env.Config.Host.MAC,
					TargetURL:       env.State.HAGrubURL,
					AuthToken:       env.State.WebhookID,
					WaitTimeSeconds: env.Config.Grub.NetworkWaitTime,
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
				env, err := GetEnv(cmd)
				if err != nil {
					return err
				}
				return performInstall(cmd, env, "")
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

			// Determine final config path from env
			env, err := GetEnv(cmd)
			if err != nil {
				return err
			}

			if dryRun {
				return doDryRun(cfg, env)
			}

			return doInstallation(cmd, cfg, env)
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

func doDryRun(cfg *config.Config, env *Env) error {
	wizard.PrintConfigSummary(os.Stdout, cfg, env.ConfigPath)

	if env.Manager == nil {
		return fmt.Errorf("no supported service manager detected")
	}

	if svcPreview, err := env.Manager.Preview(env.ConfigPath); err == nil {
		tap.Box(svcPreview, fmt.Sprintf(" %s Service Preview ", env.Manager.Name()), tap.BoxOptions{
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

func doInstallation(cmd *cobra.Command, cfg *config.Config, env *Env) error {
	if err := os.MkdirAll(filepath.Dir(env.ConfigPath), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := config.Save(cfg, env.ConfigPath); err != nil {
		return err
	}

	// Initialize empty state
	pairState := &config.State{Paired: false}
	if err := pairState.Save(env.StatePath); err != nil {
		return err
	}

	tap.Outro("Configuration setup complete.", tap.MessageOptions{
		Hint: fmt.Sprintf("saved to: %s", env.ConfigPath),
	})

	tap.Intro("Proceeding with installation...")

	// Refresh env after saving
	env, err := GetEnv(cmd)
	if err != nil {
		return err
	}

	if err := performInstall(cmd, env, ""); err != nil {
		if err == ErrElevated {
			return nil
		}
		return err
	}

	tap.Outro("Setup complete!")
	return nil
}

func performInstall(cmd *cobra.Command, env *Env, token string) error {
	log.Debug().Interface("config", env.ConfigPath).Msg("Starting installation process")

	if env.Manager == nil {
		return fmt.Errorf("no supported service manager detected")
	}

	if err := env.Manager.CheckPermissions(); err != nil {
		return err
	}

	absConfig, err := filepath.Abs(env.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	if env.Config.Daemon.ReportBootOptions {
		waitTime := config.DefaultGrubWaitSeconds
		if env.Config.Grub.NetworkWaitTime != 0 {
			waitTime = env.Config.Grub.NetworkWaitTime
		}

		targetURL := env.State.HAGrubURL
		if env.Config.Grub.URL != "" {
			targetURL = env.Config.Grub.URL
		}

		opts := grub.SetupOptions{
			TargetMAC:       env.Config.Host.MAC,
			TargetURL:       targetURL,
			AuthToken:       env.State.WebhookID,
			WaitTimeSeconds: waitTime,
		}

		warning := env.Grub.SetupWarning()
		tap.Message("Installing into grub...", tap.MessageOptions{
			Hint: warning,
		})

		if err := env.Grub.Setup(opts); err != nil {
			return fmt.Errorf("failed to install grub: %w", err)
		}

		if env.State.HADaemonURL != "" && env.State.WebhookID != "" {
			tap.Message("Pushing initial boot options to Home Assistant...")
			haClient := homeassistant.NewClient(env.State.HADaemonURL, env.State.WebhookID, nil)

			options, err := env.Grub.GetBootOptions()
			if err != nil {
				return err
			}

			if err := haClient.UpdateBootOptions(env.Config, env.State, options); err != nil {
				return err
			}
			tap.Message("Successfully pushed initial state to Home Assistant.")
		}
	}

	tap.Message(fmt.Sprintf("Installing into service manager: %s", env.Manager.Name()))
	if err := env.Manager.Configure(env.Config); err != nil {
		return fmt.Errorf("failed to configure service: %w", err)
	}

	if err := env.Manager.Install(absConfig); err != nil {
		return fmt.Errorf("failed to install manager: %w", err)
	}

	tap.Message("Starting service...")
	if err := env.Manager.Start(); err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	tap.Message("Installation completed successfully.")
	return nil
}

func IsInstalled(cmd *cobra.Command) (bool, error) {
	env, err := GetEnv(cmd)
	if err != nil {
		return false, err
	}
	if env.Manager == nil {
		return false, nil
	}
	return env.Manager.IsInstalled()
}
