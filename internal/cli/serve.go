package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/jjack/grubstation/internal/api"
	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/grub"
	"github.com/jjack/grubstation/internal/homeassistant"
	"github.com/jjack/grubstation/internal/host"
	"github.com/jjack/grubstation/internal/mdns"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the background agent service",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, cfgFile, err := GetConfig(cmd)
			if err != nil {
				return err
			}

			// Load pairing state
			state, err := config.LoadState(cfgFile)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to load pairing state")
				state = &config.State{}
			}

			if cfg.Daemon.ReportBootOptions {
				// Drift detection
				waitTime := config.DefaultGrubWaitSeconds
				if cfg.Grub.NetworkWaitTime != 0 {
					waitTime = cfg.Grub.NetworkWaitTime
				}

				targetURL := state.HAGrubURL
				if cfg.Grub.URL != "" {
					targetURL = cfg.Grub.URL
				}

				g := grub.NewGrub()
				g.ConfigPath = cfg.Grub.Path
				drift, err := g.CheckDrift(grub.SetupOptions{
					TargetMAC:       cfg.Host.MAC,
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
			mdnsServer, err := mdns.Start(cfg.Daemon.Port, cfg.Host.MAC, cfg.Host.Interface, state.Paired)
			if err != nil {
				log.Error().Err(err).Msg("Failed to start mDNS responder")
			} else {
				defer func() { _ = mdnsServer.Shutdown() }()
			}

			g := grub.NewGrub()
			g.ConfigPath = cfg.Grub.Path
			server := api.NewServer(cfg, state, cfgFile, g, host.New().GetIPInfo, mdnsServer)
			server.ShutdownHandler = func() error {
				// Perform OS-specific shutdown
				if runtime.GOOS == "windows" {
					return exec.Command("shutdown", "/s", "/t", "0").Run()
				} else {
					return exec.Command("poweroff").Run()
				}
			}

			httpSrv := &http.Server{
				Addr:         fmt.Sprintf(":%d", cfg.Daemon.Port),
				Handler:      server.Router,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  120 * time.Second,
			}

			// Initial push if paired
			if state.Paired {
				go func() {
					haClient := homeassistant.NewClient(state.HADaemonURL, state.WebhookID, nil)

					if cfg.Daemon.ReportBootOptions {
						log.Info().Msg("Pushing initial boot options to Home Assistant")
						g := grub.NewGrub()
						g.ConfigPath = cfg.Grub.Path
						options, _ := g.GetBootOptions()
						if err := haClient.UpdateBootOptions(cfg, state, options); err != nil {
							log.Error().Err(err).Msg("Initial update failed")
						}
					}
				}()
			}

			// Graceful shutdown handling
			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

			go func() {
				log.Info().Int("port", cfg.Daemon.Port).Msg("Starting HTTP server")
				if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Error().Err(err).Msg("Server failed")
					stop <- syscall.SIGTERM
				}
			}()

			<-stop

			log.Info().Msg("Shutting down...")

			// Final push if paired and enabled
			if cfg.Daemon.ReportBootOptions && state.Paired {
				haClient := homeassistant.NewClient(state.HADaemonURL, state.WebhookID, nil)

				g := grub.NewGrub()
				g.ConfigPath = cfg.Grub.Path
				options, _ := g.GetBootOptions()
				_ = haClient.UpdateBootOptions(cfg, state, options)
			}

			// We still need context for http server shutdown
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return httpSrv.Shutdown(shutdownCtx)
		},
	}
}
