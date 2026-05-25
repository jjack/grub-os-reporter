//go:build linux

package cli

import (
	"fmt"

	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/homeassistant"
	"github.com/spf13/cobra"
)

func NewBootPushCmd(deps *CommandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "push",
		Short: "Push the list of available OSes to Home Assistant",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load state for HA credentials
			state, _ := config.LoadState(deps.ConfigFile)
			if state.HADaemonURL == "" || state.WebhookID == "" {
				return fmt.Errorf("homeassistant url and webhook_id must be configured")
			}

			options, err := deps.Grub.GetBootOptions(cmd.Context())
			if err != nil {
				return err
			}

			client := homeassistant.NewClient(state.HADaemonURL, state.WebhookID, nil)
			if err := client.UpdateBootOptions(cmd.Context(), deps.Config.Host.MAC, deps.Config.Host.Address, options, deps.Config.WakeOnLan.Address, deps.Config.WakeOnLan.Port); err != nil {
				return err
			}

			cmd.Println("Successfully pushed boot options to Home Assistant")
			return nil
		},
	}
}
