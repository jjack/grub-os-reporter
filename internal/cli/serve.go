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
			env, err := GetEnv(cmd)
			if err != nil {
				return err
			}

			if env.Config.Daemon.ReportBootOptions {
				// Drift detection
				waitTime := config.DefaultGrubWaitSeconds
				if env.Config.Grub.NetworkWaitTime != 0 {
					waitTime = env.Config.Grub.NetworkWaitTime
				}

				targetURL := env.State.HAGrubURL
				if env.Config.Grub.URL != "" {
					targetURL = env.Config.Grub.URL
				}

				drift, err := env.Grub.CheckDrift(grub.SetupOptions{
					TargetMAC:       env.Config.Host.MAC,
					TargetURL:       targetURL,
					AuthToken:       env.State.WebhookID,
					WaitTimeSeconds: waitTime,
				})
				if err == nil && drift {
					log.Warn().Msg("GRUB configuration drift detected. Your installed GRUB script does not match the current config. Run 'grubstation setup --apply' to sync.")
				} else if err != nil {
					log.Debug().Err(err).Msg("Failed to check for GRUB drift")
				}
			}

			// Start mDNS
			mdnsServer, err := mdns.Start(env.Config.Daemon.Port, env.Config.Host.MAC, env.Config.Host.Interface, env.State.Paired)
			if err != nil {
				log.Error().Err(err).Msg("Failed to start mDNS responder")
			} else {
				defer func() { _ = mdnsServer.Shutdown() }()
			}

			server := api.NewServer(env.Config, env.State, env.ConfigPath, env.Grub, host.New().GetIPInfo, mdnsServer)
			server.ShutdownHandler = func() error {
				// Perform OS-specific shutdown
				return shutdownSystem()
			}

			httpSrv := &http.Server{
				Addr:         fmt.Sprintf(":%d", env.Config.Daemon.Port),
				Handler:      server.Router,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  120 * time.Second,
			}

			// Initial handshake if paired
			if env.State.Paired {
				go func() {
					haClient := homeassistant.NewClient(env.State.HADaemonURL, env.State.WebhookID, nil)

					if env.Config.Daemon.ReportBootOptions {
						log.Info().Msg("Pushing initial boot options to Home Assistant")
						options, _ := env.Grub.GetBootOptions()
						if err := haClient.UpdateBootOptions(env.Config, env.State, options); err != nil {
							log.Error().Err(err).Msg("Initial update failed")
						}
					}
				}()
			}

			// Graceful shutdown handling
			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

			go func() {
				log.Info().Int("port", env.Config.Daemon.Port).Msg("Starting HTTP server")
				if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Error().Err(err).Msg("Server failed")
					stop <- syscall.SIGTERM
				}
			}()

			<-stop

			log.Info().Msg("Shutting down...")

			// Final push if paired and enabled
			if env.Config.Daemon.ReportBootOptions && env.State.Paired {
				haClient := homeassistant.NewClient(env.State.HADaemonURL, env.State.WebhookID, nil)
				options, _ := env.Grub.GetBootOptions()
				_ = haClient.UpdateBootOptions(env.Config, env.State, options)
			}

			// We still need context for http server shutdown
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return httpSrv.Shutdown(shutdownCtx)
		},
	}
}

func shutdownSystem() error {
	log.Info().Msg("Shutting down system...")
	switch runtime.GOOS {
	case "linux":
		return exec.Command("poweroff").Run()
	case "windows":
		return exec.Command("shutdown", "/s", "/t", "0").Run()
	default:
		return fmt.Errorf("shutdown not supported on OS: %s", runtime.GOOS)
	}
}
