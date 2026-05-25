package cli

import (
	"github.com/rs/zerolog/log"

	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/daemon"
	"github.com/jjack/grubstation/internal/grub"
	"github.com/jjack/grubstation/internal/homeassistant"
	"github.com/jjack/grubstation/internal/host"
	"github.com/jjack/grubstation/internal/version"
	"github.com/spf13/cobra"
)

func NewServeCmd(deps *CommandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the persistent agent daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, _ := deps.Manager(cmd.Context())
			mgrName := ""
			if activeMgr := mgr; activeMgr != nil {
				mgrName = activeMgr.Name()
			}

			if deps.Config.Daemon.ReportBootOptions {
				// Drift detection
				waitTime := config.DefaultGrubWaitSeconds
				if deps.Config.Grub.NetworkWaitTime != 0 {
					waitTime = deps.Config.Grub.NetworkWaitTime
				}

				// Get HA details from state for drift check if needed
				state, _ := config.LoadState(deps.ConfigFile)
				targetURL := state.HADaemonURL
				if deps.Config.Grub.URL != "" {
					targetURL = deps.Config.Grub.URL
				}

				drift, err := deps.Grub.CheckDrift(grub.SetupOptions{
					TargetMAC:       deps.Config.Host.MAC,
					TargetURL:       targetURL,
					AuthToken:       state.WebhookID,
					WaitTimeSeconds: waitTime,
				})
				if err == nil && drift {
					log.Warn().Msg("GRUB configuration drift detected. Your installed GRUB script does not match the current config. Run 'grubstation setup --apply' to sync.")
				} else if err != nil {
					log.Debug().Err(err).Msg("Failed to check for GRUB drift")
				}
			}

			// Load and apply pairing state (takes precedence over config.yaml)
			state, err := config.LoadState(deps.ConfigFile)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to load pairing state")
			} else if state.WebhookID != "" {
				log.Info().Msg("Applying persistent pairing state")
				// We don't overwrite Config anymore, we use State directly for HA
			}

			var haClient *homeassistant.Client
			if state.HADaemonURL != "" && state.WebhookID != "" {
				haClient = homeassistant.NewClient(state.HADaemonURL, state.WebhookID, nil)
			}

			d := daemon.New(daemon.Config{
				Port:                deps.Config.Daemon.Port,
				ReportBootOptions:   deps.Config.Daemon.ReportBootOptions,
				APIKey:              state.APIKey, // Use key from state
				MACAddress:          deps.Config.Host.MAC,
				HostAddress:         deps.Config.Host.Address,
				WolBroadcastAddress: deps.Config.WakeOnLan.Address,
				WolBroadcastPort:    deps.Config.WakeOnLan.Port,
			}, daemon.Metadata{
				OS:             host.Platform(),
				Version:        version.Version,
				ServiceManager: mgrName,
			}, deps.Grub, haClient)

			d.OnPair = func(req daemon.PairRequest) error {
				s := &config.State{
					Paired:      true,
					WebhookID:   req.WebhookID,
					APIKey:      req.APIKey,
					HADaemonURL: req.HADaemonURL,
					HAGrubURL:   req.HAGrubURL,
				}
				return s.Save(deps.ConfigFile)
			}

			d.OnUnpair = func() error {
				s := &config.State{Paired: false}
				return s.Save(deps.ConfigFile)
			}

			return d.Run(cmd.Context())
		},
	}
}
