package cli

import (
	"fmt"
	"os"

	"github.com/jjack/grubstation/internal/config"
	"github.com/spf13/cobra"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage the grubstation configuration",
	}

	cmd.AddCommand(NewConfigValidateCmd())
	cmd.AddCommand(NewConfigInitCmd())

	return cmd
}

func NewConfigInitCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a default configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(output); err == nil {
				return fmt.Errorf("config file already exists: %s", output)
			}

			cfg := &config.Config{
				Host: config.HostConfig{
					Interface: "eth0",
					MAC:       "00:00:00:00:00:00",
				},
				WakeOnLan: config.WakeOnLanConfig{
					Address: config.DefaultWolBroadcastAddress,
					Port:    config.DefaultWolBroadcastPort,
				},
				Daemon: config.DaemonConfig{
					Port:              config.DefaultAgentPort,
					ReportBootOptions: true,
					APIKey:            "REPLACE_ME_OR_LEAVE_EMPTY_FOR_TOFU",
				},
				Grub: config.GrubConfig{
					Path:            "/boot/grub/grub.cfg",
					NetworkWaitTime: 15,
					URL:             "http://homeassistant.local:8123",
				},
			}

			if err := config.SaveExhaustive(cfg, output); err != nil {
				return err
			}

			cmd.Printf("Default configuration generated at: %s\n", output)
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "config.yaml", "Path to write the default configuration")

	return cmd
}

func NewConfigValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate an existing configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := GetEnv(cmd)
			if err != nil {
				return err
			}
			cmd.Println("Configuration is valid.")
			return nil
		},
	}
}
