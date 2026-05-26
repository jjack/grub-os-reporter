//go:build linux

package cli

import (
	"fmt"

	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/grub"
	"github.com/jjack/grubstation/internal/homeassistant"
	"github.com/spf13/cobra"
)

func NewBootPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push",
		Short: "Push the list of available OSes to Home Assistant",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load state for HA credentials
			state, _ := config.LoadState(GlobalConfigFile)
			if state.HADaemonURL == "" || state.WebhookID == "" {
				return fmt.Errorf("homeassistant url and webhook_id must be configured")
			}

			g := grub.NewGrub()
			g.ConfigPath = GlobalConfig.Grub.Path
			options, err := g.GetBootOptions()
			if err != nil {
				return err
			}

			client := homeassistant.NewClient(state.HADaemonURL, state.WebhookID, nil)
			if err := client.UpdateBootOptions(GlobalConfig, state, options); err != nil {
				return err
			}

			cmd.Println("Successfully pushed boot options to Home Assistant")
			return nil
		},
	}
}
