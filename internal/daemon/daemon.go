package daemon

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/jjack/grubstation/internal/grub"
	"github.com/jjack/grubstation/internal/homeassistant"
)

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Config holds the daemon configuration.
type Config struct {
	Port              int
	ReportBootOptions bool
	APIKey            string
	RetryInterval     time.Duration
	ShutdownDelay     time.Duration

	// Reporting fields
	MACAddress          string
	HostAddress         string
	WolBroadcastAddress string
	WolBroadcastPort    int
}

// Metadata holds system information.
type Metadata struct {
	OS             string `json:"os"`
	Version        string `json:"version"`
	ServiceManager string `json:"service_manager"`
}

// Daemon represents the background service.
type Daemon struct {
	Config          Config
	Metadata        Metadata
	Grub            *grub.Grub
	HAClient        *homeassistant.Client
	ShutdownHandler func() error

	mu sync.Mutex
}

func New(cfg Config, meta Metadata, g *grub.Grub, haClient *homeassistant.Client) *Daemon {
	return &Daemon{
		Config:   cfg,
		Metadata: meta,
		Grub:     g,
		HAClient: haClient,
		ShutdownHandler: func() error {
			return shutdownSystem()
		},
	}
}

// TriggerUpdate performs a boot options push to Home Assistant.
func (d *Daemon) TriggerUpdate(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.Config.ReportBootOptions {
		return nil
	}

	if d.HAClient == nil {
		return fmt.Errorf("home assistant client not configured")
	}

	var bootOptions []string
	if d.Grub != nil {
		var err error
		bootOptions, err = d.Grub.GetBootOptions(ctx)
		if err != nil {
			return fmt.Errorf("failed to get boot options: %w", err)
		}
	}

	log.Debug().Msg("Triggering boot options update to Home Assistant")
	if err := d.HAClient.UpdateBootOptions(ctx, d.Config.MACAddress, d.Config.HostAddress, bootOptions, d.Config.WolBroadcastAddress, d.Config.WolBroadcastPort); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	log.Debug().Msg("Update successful")
	return nil
}

// run contains the core daemon lifecycle.
func (d *Daemon) run(ctx context.Context) error {
	token, err := d.ensureToken()
	if err != nil {
		return err
	}

	// 1. Initial Handshake (Register + First Update)
	if err := d.performInitialHandshake(ctx, token); err != nil {
		return err
	}

	// 2. Start Listeners
	go d.listenUnixSocket(ctx, token)

	srv := d.newHTTPServer(token)
	go func() {
		log.Info().Interface("port", d.Config.Port).Msg("Starting HTTP listener")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("HTTP server failed")
		}
	}()

	log.Info().Msg("Daemon is running and waiting for termination")

	// 3. Wait for context cancellation and cleanup
	<-ctx.Done()
	return d.cleanup(srv)
}

func (d *Daemon) ensureToken() (string, error) {
	token := d.Config.APIKey
	if token == "" {
		generated, err := generateToken()
		if err != nil {
			return "", fmt.Errorf("failed to generate dynamic token: %w", err)
		}
		log.Info().Msg("Using dynamically generated TOFU token")
		return generated, nil
	}
	log.Info().Msg("Using configured API key")
	return token, nil
}

func (d *Daemon) performInitialHandshake(ctx context.Context, token string) error {
	if d.HAClient == nil {
		return nil
	}

	backoff := d.Config.RetryInterval
	if backoff == 0 {
		backoff = 5 * time.Second
	}
	maxBackoff := 5 * time.Minute

	log.Info().Msg("Starting initial registration with Home Assistant")
	for {
		if err := d.HAClient.RegisterAgent(ctx, d.Config.MACAddress, d.Config.HostAddress, token, d.Config.Port); err != nil {
			log.Warn().Err(err).Interface("retry_in", backoff).Msg("Initial registration failed, retrying...")
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		log.Info().Msg("Initial registration successful")
		break
	}

	if err := d.TriggerUpdate(ctx); err != nil {
		log.Error().Msg("Initial update failed")
	} else {
		log.Info().Msg("Initial update successful")
	}

	return nil
}

func (d *Daemon) newHTTPServer(token string) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		status := struct {
			Status string `json:"status"`
			Metadata
		}{
			Status:   "ok",
			Metadata: d.Metadata,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(status)
	})

	mux.HandleFunc("POST /shutdown", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+token {
			log.Warn().Interface("remote_addr", r.RemoteAddr).Msg("Unauthorized shutdown request")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "Forbidden"})
			return
		}

		log.Info().Msg("Shutdown requested via HTTP")

		// Perform pre-shutdown push (synchronous)
		if err := d.TriggerUpdate(r.Context()); err != nil {
			log.Error().Err(err).Msg("Pre-shutdown push failed")
		}

		if err := d.performOSShutdown(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", d.Config.Port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      mux,
	}
}

func (d *Daemon) cleanup(srv *http.Server) error {
	log.Info().Msg("Shutting down daemon...")

	if runtime.GOOS == "linux" {
		log.Info().Msg("Performing final GRUB report push")
		pushCtx, pushCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer pushCancel()
		if err := d.TriggerUpdate(pushCtx); err != nil {
			log.Error().Err(err).Msg("Final push failed")
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func (d *Daemon) performOSShutdown() error {
	if d.ShutdownHandler != nil {
		return d.ShutdownHandler()
	}
	return nil
}
