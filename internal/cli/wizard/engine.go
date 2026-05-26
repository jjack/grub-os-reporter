package wizard

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"charm.land/huh/v2"
	"github.com/jjack/grubstation/internal/config"
	"github.com/rs/zerolog/log"
)

// BuildIfaceOptions builds the selection options for network interfaces.
func BuildIfaceOptions(interfaces []net.Interface, ipProvider func(net.Interface) ([]string, map[string]string)) []huh.Option[int] {
	var opts []huh.Option[int]
	for i, inf := range interfaces {
		ips, _ := ipProvider(inf)
		label := fmt.Sprintf("%s (%s) [%s]", inf.Name, inf.HardwareAddr.String(), strings.Join(ips, ", "))
		opts = append(opts, huh.NewOption(label, i))
	}
	return opts
}

// BuildWolOptions builds the selection options for the WOL broadcast address.
func BuildWolOptions(ips []string, ipBroadcasts map[string]string) []huh.Option[string] {
	opts := []huh.Option[string]{
		huh.NewOption(fmt.Sprintf("%s (Default)", config.DefaultWolBroadcastAddress), config.DefaultWolBroadcastAddress),
	}

	seenBroadcasts := make(map[string]bool)
	for _, ip := range ips {
		bc, ok := ipBroadcasts[ip]
		if !ok {
			continue
		}

		if !seenBroadcasts[bc] {
			seenBroadcasts[bc] = true
			label := fmt.Sprintf("%s (Subnet broadcast for %s)", bc, ip)
			opts = append(opts, huh.NewOption(label, bc))
		}
	}
	return opts
}

// ValidatePort checks if a port is valid and available.
func ValidatePort(s string, isReinstall bool, currentPort int, portChecker func(int) error) error {
	if err := config.ValidatePort(s); err != nil {
		return err
	}
	port, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	if isReinstall && port == currentPort {
		return nil
	}

	return portChecker(port)
}

// CheckPortAvailability is the default implementation of the port checker.
func CheckPortAvailability(port int) error {
	log.Debug().Interface("port", port).Msg("Checking port availability")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Debug().Interface("port", port).Err(err).Msg("Port availability check failed")
		return fmt.Errorf("port %d is in use or unavailable: %v", port, err)
	}
	_ = listener.Close()
	log.Debug().Interface("port", port).Msg("Port is available")
	return nil
}
