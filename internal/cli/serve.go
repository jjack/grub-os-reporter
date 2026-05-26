package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jjack/grubstation/internal/api"
	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/grub"
	"github.com/jjack/grubstation/internal/homeassistant"
	"github.com/jjack/grubstation/internal/mdns"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewServeCmd(deps *CommandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the background agent service",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load pairing state
			state, err := config.LoadState(deps.ConfigFile)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to load pairing state")
				state = &config.State{}
			}

			if deps.Config.Daemon.ReportBootOptions {
				// Drift detection
				waitTime := config.DefaultGrubWaitSeconds
				if deps.Config.Grub.NetworkWaitTime != 0 {
					waitTime = deps.Config.Grub.NetworkWaitTime
				}

				targetURL := state.HAGrubURL
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

			// Start mDNS
			mdnsServer, err := mdns.Start(deps.Config.Daemon.Port, deps.Config.Host.MAC, deps.Config.Host.Interface, state.Paired)
			if err != nil {
				log.Error().Err(err).Msg("Failed to start mDNS responder")
			} else {
				defer mdnsServer.Shutdown()
			}

			server := api.NewServer(deps.Config, state, deps.ConfigFile, deps.Grub, deps.Host.GetIPInfo, mdnsServer)
			server.ShutdownHandler = func() error {
				// Perform OS-specific shutdown
				return shutdownSystem()
			}

			httpSrv := &http.Server{
				Addr:         fmt.Sprintf(":%d", deps.Config.Daemon.Port),
				Handler:      server.Router,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  120 * time.Second,
			}

			// Initial handshake if paired
			if state.Paired {
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()

					haClient := homeassistant.NewClient(state.HADaemonURL, state.WebhookID, nil)
					log.Info().Msg("Performing initial registration with Home Assistant")

					// Get current IP for this interface to register with HA
					addr := ""
					if iface, err := net.InterfaceByName(deps.Config.Host.Interface); err == nil {
						ips, _ := deps.Host.GetIPInfo(*iface)
						if len(ips) > 0 {
							addr = ips[0]
						}
					}

					if err := haClient.RegisterAgent(ctx, deps.Config.Host.MAC, addr, state.APIKey, deps.Config.Daemon.Port); err != nil {
						log.Warn().Err(err).Msg("Initial registration failed")
					}

					if deps.Config.Daemon.ReportBootOptions {
						log.Info().Msg("Pushing initial boot options to Home Assistant")
						options, _ := deps.Grub.GetBootOptions(ctx)
						if err := haClient.UpdateBootOptions(ctx, deps.Config.Host.MAC, addr, options, deps.Config.WakeOnLan.Address, deps.Config.WakeOnLan.Port); err != nil {
							log.Error().Err(err).Msg("Initial update failed")
						}
					}
				}()
			}

			// Graceful shutdown handling
			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

			go func() {
				log.Info().Int("port", deps.Config.Daemon.Port).Msg("Starting HTTP server")
				if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Error().Err(err).Msg("Server failed")
					stop <- syscall.SIGTERM
				}
			}()

			<-stop

			log.Info().Msg("Shutting down...")

			// Final push if paired and enabled
			if deps.Config.Daemon.ReportBootOptions && state.Paired {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				haClient := homeassistant.NewClient(state.HADaemonURL, state.WebhookID, nil)

				// Get current IP
				addr := ""
				if iface, err := net.InterfaceByName(deps.Config.Host.Interface); err == nil {
					ips, _ := deps.Host.GetIPInfo(*iface)
					if len(ips) > 0 {
						addr = ips[0]
					}
				}

				options, _ := deps.Grub.GetBootOptions(ctx)
				_ = haClient.UpdateBootOptions(ctx, deps.Config.Host.MAC, addr, options, deps.Config.WakeOnLan.Address, deps.Config.WakeOnLan.Port)
			}

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return httpSrv.Shutdown(shutdownCtx)
		},
	}
}
