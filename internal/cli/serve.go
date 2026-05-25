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
				targetURL := deps.Config.HomeAssistant.URL
				if deps.Config.Grub != nil {
					if deps.Config.Grub.WaitTimeSeconds != 0 {
						waitTime = deps.Config.Grub.WaitTimeSeconds
					}
					if deps.Config.Grub.URL != "" {
						targetURL = deps.Config.Grub.URL
					}
				}

				drift, err := deps.Grub.CheckDrift(grub.SetupOptions{
					TargetMAC:       deps.Config.Host.MACAddress,
					TargetURL:       targetURL,
					AuthToken:       deps.Config.HomeAssistant.WebhookID,
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
				deps.Config.HomeAssistant.WebhookID = state.WebhookID
				deps.Config.HomeAssistant.URL = state.HADaemonURL
				deps.Config.Daemon.APIKey = state.APIKey
				if deps.Config.Grub == nil {
					deps.Config.Grub = &config.GrubConfig{}
				}
				deps.Config.Grub.URL = state.HAGrubURL
			}

			var haClient *homeassistant.Client
			if deps.Config.HomeAssistant.URL != "" && deps.Config.HomeAssistant.WebhookID != "" {
				haClient = homeassistant.NewClient(deps.Config.HomeAssistant.URL, deps.Config.HomeAssistant.WebhookID, nil)
			}

			var wolAddr string
			var wolPort int
			if deps.Config.WakeOnLan != nil {
				wolAddr = deps.Config.WakeOnLan.Address
				wolPort = deps.Config.WakeOnLan.Port
			}

			d := daemon.New(daemon.Config{
				Port:                deps.Config.Daemon.Port,
				ReportBootOptions:   deps.Config.Daemon.ReportBootOptions,
				APIKey:              deps.Config.Daemon.APIKey,
				MACAddress:          deps.Config.Host.MACAddress,
				HostAddress:         deps.Config.Host.Address,
				WolBroadcastAddress: wolAddr,
				WolBroadcastPort:    wolPort,
			}, daemon.Metadata{
				OS:             host.Platform(),
				Version:        version.Version,
				ServiceManager: mgrName,
			}, deps.Grub, haClient)

			d.OnPair = func(req daemon.PairRequest) error {
				s := &config.State{
					WebhookID:   req.WebhookID,
					APIKey:      req.APIKey,
					HADaemonURL: req.HADaemonURL,
					HAGrubURL:   req.HAGrubURL,
				}
				return config.SaveState(deps.ConfigFile, s)
			}

			d.OnUnpair = func() error {
				return config.SaveState(deps.ConfigFile, &config.State{})
			}

			return d.Run(cmd.Context())
		},
	}
}
