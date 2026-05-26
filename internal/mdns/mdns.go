package mdns

import (
	"context"
	"fmt"
	"os"

	"github.com/brutella/dnssd"
	"github.com/rs/zerolog/log"
)

type Server struct {
	responder dnssd.Responder
	handle    dnssd.ServiceHandle
	cancel    context.CancelFunc
	mac       string
}

func Start(port int, mac string, ifaceName string, paired bool) (*Server, error) {
	host, _ := os.Hostname()

	cfg := dnssd.Config{
		Name: host,
		Type: "_grubstation._tcp",
		Port: port,
		Text: map[string]string{
			"mac":    mac,
			"paired": fmt.Sprintf("%v", paired),
		},
	}

	if ifaceName != "" {
		log.Debug().Str("iface", ifaceName).Msg("Binding mDNS to configured interface")
		cfg.Ifaces = []string{ifaceName}
	}

	sv, err := dnssd.NewService(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create mDNS service: %w", err)
	}

	rp, err := dnssd.NewResponder()
	if err != nil {
		return nil, fmt.Errorf("failed to create mDNS responder: %w", err)
	}

	hdl, err := rp.Add(sv)
	if err != nil {
		return nil, fmt.Errorf("failed to add service to mDNS responder: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		log.Debug().Msg("mDNS responder started")
		if err := rp.Respond(ctx); err != nil && err != context.Canceled {
			log.Error().Err(err).Msg("mDNS responder error")
		}
	}()

	return &Server{
		responder: rp,
		handle:    hdl,
		cancel:    cancel,
		mac:       mac,
	}, nil
}

func (s *Server) Shutdown() error {
	log.Debug().Msg("Shutting down mDNS responder")
	s.cancel()
	return nil
}

func (s *Server) UpdatePaired(paired bool) error {
	newText := map[string]string{
		"mac":    s.mac,
		"paired": fmt.Sprintf("%v", paired),
	}

	log.Debug().
		Interface("txt", newText).
		Msg("Updating mDNS broadcast TXT records")

	s.handle.UpdateText(newText, s.responder)
	return nil
}
