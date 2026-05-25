//go:build linux

package cli

import (
	"fmt"

	"github.com/jjack/grubstation/internal/daemon"
	"github.com/jjack/grubstation/internal/homeassistant"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewBootPushCmd(deps *CommandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "push",
		Short: "Push the list of available OSes to Home Assistant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := daemon.RequestPushViaSocket(cmd.Context()); err == nil {
				cmd.Println("Successfully pushed boot options to Home Assistant (via running daemon)")
				return nil
			} else {
				log.Debug().Err(err).Msg("Could not push via daemon socket, falling back to direct push")
			}

			if deps.Config.HomeAssistant.URL == "" || deps.Config.HomeAssistant.WebhookID == "" {
				return fmt.Errorf("homeassistant url and webhook_id must be configured")
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

			client := homeassistant.NewClient(deps.Config.HomeAssistant.URL, deps.Config.HomeAssistant.WebhookID, nil)
			if err := client.UpdateBootOptions(cmd.Context(), deps.Config.Host.MACAddress, deps.Config.Host.Address, options, wolAddr, wolPort); err != nil {
				return err
			}

			cmd.Println("Successfully pushed boot options to Home Assistant")
			return nil
		},
	}
}
