package cli

import (
	"fmt"
	"io"
	"net"
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
		Short: "Manage the GrubStation background service",
	}

	cmd.AddCommand(NewServiceInstallCmd(deps))
	cmd.AddCommand(NewServiceRemoveCmd(deps))
	cmd.AddCommand(NewServiceStartCmd(deps))
	cmd.AddCommand(NewServiceStopCmd(deps))
	cmd.AddCommand(NewServiceStatusCmd(deps))

	return cmd
}

func NewServiceInstallCmd(deps *CommandDeps) *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the agent as a system service",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := deps.Manager(cmd.Context())
			if err != nil {
				return err
			}

			if err := mgr.CheckPermissions(cmd.Context()); err != nil {
				return err
			}

			cmd.Printf("Installing service: %s\n", mgr.Name())
			if err := mgr.Install(cmd.Context(), configPath); err != nil {
				return err
			}

			cmd.Println("Service installed successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", config.DefaultConfigPath(), "Path to configuration file")

	return cmd
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
				addr := ""

				// Get current IP for this interface to register with HA
				if iface, err := net.InterfaceByName(deps.Config.Host.Interface); err == nil {
					ips, _ := deps.Host.GetIPInfo(*iface)
					if len(ips) > 0 {
						addr = ips[0]
					}
				}

				if mac != "" && addr != "" {
					cmd.Printf("Unregistering from Home Assistant...\n")
					client := homeassistant.NewClient(state.HADaemonURL, state.WebhookID, nil)
					if err := client.UnregisterHost(cmd.Context(), mac, addr); err != nil {
						cmd.Printf("Warning: failed to unregister from Home Assistant: %v\n", err)
					}
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
		Short: "Start the system service",
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
		Short: "Stop the system service",
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
		Short: "Check the service status",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := deps.Manager(cmd.Context())
			if err != nil {
				return err
			}

			active := mgr.IsActive(cmd.Context())
			status := "Inactive"
			if active {
				status = "Active"
			}

			cmd.Printf("Service name: %s\n", mgr.Name())
			cmd.Printf("Service status: %s\n", status)

			// Try to check local daemon status if running
			client := &http.Client{Timeout: 1 * time.Second}
			resp, err := client.Get(fmt.Sprintf("http://localhost:%d/status", deps.Config.Daemon.Port))
			if err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				cmd.Printf("Daemon status: %s\n", string(body))
			} else {
				cmd.Printf("Daemon status check returned non-OK status: %v\n", err)
			}

			return nil
		},
	}
}
