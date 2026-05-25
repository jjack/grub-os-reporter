package api

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/grub"
	"github.com/jjack/grubstation/internal/homeassistant"
	"github.com/jjack/grubstation/internal/version"
	"github.com/rs/zerolog/log"
)

type Server struct {
	Config     *config.Config
	State      *config.State
	ConfigFile string
	Grub       *grub.Grub
	Router     *chi.Mux

	ShutdownHandler func() error
}

func NewServer(cfg *config.Config, state *config.State, configFile string, g *grub.Grub) *Server {
	s := &Server{
		Config:     cfg,
		State:      state,
		ConfigFile: configFile,
		Grub:       g,
		Router:     chi.NewRouter(),
	}

	s.Router.Use(ZerologMiddleware)
	s.Router.Use(middleware.Recoverer)

	s.Router.Get("/status", s.handleStatus)
	s.Router.Post("/pair", s.handlePair)

	s.Router.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Post("/unpair", s.handleUnpair)
		r.Post("/shutdown", s.handleShutdown)
	})

	return s
}

func (s *Server) jsonResponse(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func (s *Server) jsonError(w http.ResponseWriter, code int, message string) {
	s.jsonResponse(w, code, map[string]string{"error": message})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+s.State.APIKey {
			log.Error().Str("remote_addr", r.RemoteAddr).Msg("Unauthorized request")
			s.jsonError(w, http.StatusForbidden, "Forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"os":      runtime.GOOS,
		"version": version.Version,
		"paired":  s.State.Paired,
	})
}

type PairRequest struct {
	WebhookID   string `json:"webhook_id"`
	APIKey      string `json:"api_key"`
	HADaemonURL string `json:"ha_daemon_url"`
	HAGrubURL   string `json:"ha_grub_url"`
	ApplyConfig bool   `json:"apply_config"`
}

func (s *Server) handlePair(w http.ResponseWriter, r *http.Request) {
	var req PairRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.WebhookID == "" || req.APIKey == "" || req.HADaemonURL == "" {
		s.jsonError(w, http.StatusBadRequest, "Missing required fields")
		return
	}

	log.Info().Msg("Pairing request received")

	s.State.Paired = true
	s.State.WebhookID = req.WebhookID
	s.State.APIKey = req.APIKey
	s.State.HADaemonURL = req.HADaemonURL
	s.State.HAGrubURL = req.HAGrubURL

	if err := s.State.Save(s.ConfigFile); err != nil {
		log.Error().Err(err).Msg("Failed to save state during pairing")
		s.jsonError(w, http.StatusInternalServerError, "Failed to persist pairing state")
		return
	}

	// Trigger initial handshake/push in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		haClient := homeassistant.NewClient(s.State.HADaemonURL, s.State.WebhookID, nil)

		// 1. Register agent
		if err := haClient.RegisterAgent(ctx, s.Config.Host.MAC, s.Config.Host.Address, s.State.APIKey, s.Config.Daemon.Port); err != nil {
			log.Warn().Err(err).Msg("Failed to register agent after pairing")
		}

		// 2. Push initial boot options if enabled
		if s.Config.Daemon.ReportBootOptions {
			options, _ := s.Grub.GetBootOptions(ctx)
			if err := haClient.UpdateBootOptions(ctx, s.Config.Host.MAC, s.Config.Host.Address, options, s.Config.WakeOnLan.Address, s.Config.WakeOnLan.Port); err != nil {
				log.Error().Err(err).Msg("Failed to push initial boot options after pairing")
			}
		}

		// 3. Apply GRUB config if requested
		if req.ApplyConfig && runtime.GOOS == "linux" {
			log.Info().Msg("Applying GRUB configuration as requested by pairing")
			err := s.Grub.Setup(ctx, grub.SetupOptions{
				TargetMAC:       s.Config.Host.MAC,
				TargetURL:       s.State.HAGrubURL,
				AuthToken:       s.State.WebhookID,
				WaitTimeSeconds: s.Config.Grub.NetworkWaitTime,
			})
			if err != nil {
				log.Error().Err(err).Msg("Failed to apply GRUB config after pairing")
			}
		}
	}()

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleUnpair(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Unpairing requested")

	s.State.Paired = false
	s.State.WebhookID = ""
	s.State.APIKey = ""
	s.State.HADaemonURL = ""
	s.State.HAGrubURL = ""

	if err := s.State.Save(s.ConfigFile); err != nil {
		log.Error().Err(err).Msg("Failed to clear state during unpairing")
		s.jsonError(w, http.StatusInternalServerError, "Failed to clear pairing state")
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Shutdown requested via HTTP")

	// Final push to HA if enabled
	if s.Config.Daemon.ReportBootOptions && s.State.Paired {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		haClient := homeassistant.NewClient(s.State.HADaemonURL, s.State.WebhookID, nil)
		options, _ := s.Grub.GetBootOptions(ctx)
		_ = haClient.UpdateBootOptions(ctx, s.Config.Host.MAC, s.Config.Host.Address, options, s.Config.WakeOnLan.Address, s.Config.WakeOnLan.Port)
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})

	go func() {
		time.Sleep(1 * time.Second) // Give response time to send
		if s.ShutdownHandler != nil {
			if err := s.ShutdownHandler(); err != nil {
				log.Error().Err(err).Msg("Shutdown handler failed")
			}
		}
	}()
}
