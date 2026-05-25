package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/homeassistant"
	"github.com/spf13/cobra"
)

func NewServiceCmd(deps *CommandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage the grubstation service",
	}

	cmd.AddCommand(NewServiceInstallCmd(deps))
	cmd.AddCommand(NewServiceRemoveCmd(deps))
	cmd.AddCommand(NewServiceStartCmd(deps))
	cmd.AddCommand(NewServiceStopCmd(deps))
	cmd.AddCommand(NewServiceStatusCmd(deps))

	return cmd
}

func NewServiceInstallCmd(deps *CommandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install the grubstation service",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := deps.Manager(cmd.Context())
			if err != nil {
				return err
			}

			cfgFile, err := cmd.Flags().GetString("config")
			if err != nil {
				return fmt.Errorf("failed to get config flag: %w", err)
			}
			absConfig, err := filepath.Abs(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to resolve config path: %w", err)
			}

			cmd.Printf("Installing service: %s\n", mgr.Name())
			if err := mgr.Install(cmd.Context(), absConfig); err != nil {
				return fmt.Errorf("failed to install manager: %w", err)
			}

			cmd.Println("Installation completed successfully.")
			return nil
		},
	}
}

func NewServiceRemoveCmd(deps *CommandDeps) *cobra.Command {
	var purge bool

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Uninstall the grubstation service and GRUB hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := deps.Manager(cmd.Context())
			if err != nil {
				return err
			}

			// Try to unregister from HA first if configured
			state, _ := config.LoadState(deps.ConfigFile)
			if state.HADaemonURL != "" && state.WebhookID != "" {
				mac := deps.Config.Host.MAC
				addr := deps.Config.Host.Address

				if mac == "" || addr == "" {
					if ifaces, err := deps.Host.GetWOLInterfaces(); err == nil && len(ifaces) > 0 {
						if mac == "" {
							mac = ifaces[0].HardwareAddr.String()
						}
						if addr == "" {
							ips, _ := deps.Host.GetIPInfo(ifaces[0])
							if len(ips) > 0 {
								addr = ips[0]
							}
						}
					}
				}

				cmd.Printf("Unregistering from Home Assistant...\n")
				client := homeassistant.NewClient(state.HADaemonURL, state.WebhookID, nil)
				if err := client.UnregisterHost(cmd.Context(), mac, addr); err != nil {
					cmd.Printf("Warning: failed to unregister from Home Assistant: %v\n", err)
				}
			}

			cmd.Printf("Removing service: %s\n", mgr.Name())
			if err := mgr.Uninstall(cmd.Context()); err != nil {
				return fmt.Errorf("failed to remove manager: %w", err)
			}

			if deps.Config.Daemon.ReportBootOptions {
				cmd.Printf("Removing GRUB hooks...\n")
				if err := deps.Grub.Uninstall(cmd.Context()); err != nil {
					return fmt.Errorf("failed to uninstall grub: %w", err)
				}
			}

			if purge {
				cfgDir := filepath.Dir(deps.ConfigFile)
				cmd.Printf("Purging configuration: %s\n", cfgDir)
				if err := os.RemoveAll(cfgDir); err != nil {
					return fmt.Errorf("failed to purge configuration: %w", err)
				}
			}

			cmd.Println("Removal completed successfully.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&purge, "purge", false, "Remove configuration files and directory")

	return cmd
}

func NewServiceStartCmd(deps *CommandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the grubstation service",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := deps.Manager(cmd.Context())
			if err != nil {
				return err
			}
			return mgr.Start(cmd.Context())
		},
	}
}

func NewServiceStopCmd(deps *CommandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the grubstation service",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := deps.Manager(cmd.Context())
			if err != nil {
				return err
			}
			return mgr.Stop(cmd.Context())
		},
	}
}

func NewServiceStatusCmd(deps *CommandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check the status of the grubstation service",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := deps.Manager(cmd.Context())
			if err != nil {
				return err
			}

			if mgr.IsActive(cmd.Context()) {
				cmd.Printf("Service %s is active\n", mgr.Name())
			} else {
				cmd.Printf("Service %s is inactive\n", mgr.Name())
			}

			// Also check health endpoint
			client := &http.Client{Timeout: 2 * time.Second}
			url := fmt.Sprintf("http://localhost:%d/status", deps.Config.Daemon.Port)
			resp, err := client.Get(url)
			if err != nil {
				cmd.Printf("Daemon status check failed: %v (daemon might not be running or port is blocked)\n", err)
				return nil
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				cmd.Printf("Daemon status: %s\n", string(body))
			} else {
				cmd.Printf("Daemon status check returned non-OK status: %d\n", resp.StatusCode)
			}

			return nil
		},
	}
}
