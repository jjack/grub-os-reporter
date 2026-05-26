package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

func NewServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage the GrubStation background service",
	}

	cmd.AddCommand(NewServiceInstallCmd())
	cmd.AddCommand(NewServiceRemoveCmd())
	cmd.AddCommand(NewServiceStartCmd())
	cmd.AddCommand(NewServiceStopCmd())
	cmd.AddCommand(NewServiceStatusCmd())

	return cmd
}

func NewServiceInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install the agent as a system service",
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := GetEnv(cmd)
			if err != nil {
				return err
			}

			if env.Manager == nil {
				return fmt.Errorf("no supported service manager detected")
			}

			if err := env.Manager.CheckPermissions(); err != nil {
				return err
			}

			cmd.Printf("Installing service: %s\n", env.Manager.Name())
			if err := env.Manager.Install(env.ConfigPath); err != nil {
				return err
			}

			cmd.Println("Service installed successfully.")
			return nil
		},
	}
}

func NewServiceRemoveCmd() *cobra.Command {
	var purge bool

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Uninstall the grubstation service and GRUB hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := GetEnv(cmd)
			if err != nil {
				return err
			}

			if env.Manager == nil {
				return fmt.Errorf("no supported service manager detected")
			}

			cmd.Printf("Removing service: %s\n", env.Manager.Name())
			if err := env.Manager.Uninstall(); err != nil {
				return fmt.Errorf("failed to remove manager: %w", err)
			}

			if env.Config.Daemon.ReportBootOptions {
				cmd.Printf("Removing GRUB hooks...\n")
				if err := env.Grub.Uninstall(); err != nil {
					return fmt.Errorf("failed to uninstall grub: %w", err)
				}
			}

			if purge {
				cfgDir := filepath.Dir(env.ConfigPath)
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

func NewServiceStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the system service",
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := GetEnv(cmd)
			if err != nil {
				return err
			}
			if env.Manager == nil {
				return fmt.Errorf("no supported service manager detected")
			}
			return env.Manager.Start()
		},
	}
}

func NewServiceStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the system service",
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := GetEnv(cmd)
			if err != nil {
				return err
			}
			if env.Manager == nil {
				return fmt.Errorf("no supported service manager detected")
			}
			return env.Manager.Stop()
		},
	}
}

func NewServiceStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check the service status",
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := GetEnv(cmd)
			if err != nil {
				return err
			}
			if env.Manager == nil {
				return fmt.Errorf("no supported service manager detected")
			}

			active := env.Manager.IsActive()
			status := "Inactive"
			if active {
				status = "Active"
			}

			cmd.Printf("Service name: %s\n", env.Manager.Name())
			cmd.Printf("Service status: %s\n", status)

			// Try to check local daemon status if running
			client := &http.Client{Timeout: 1 * time.Second}
			resp, err := client.Get(fmt.Sprintf("http://localhost:%d/status", env.Config.Daemon.Port))
			if err == nil {
				defer func() { _ = resp.Body.Close() }()
				body, _ := io.ReadAll(resp.Body)
				cmd.Printf("Daemon status: %s\n", string(body))
			} else {
				cmd.Printf("Daemon status check returned non-OK status: %v\n", err)
			}

			return nil
		},
	}
}
