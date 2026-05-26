package wizard

import (
	"fmt"
	"net"
	"strings"

	"charm.land/huh/v2"
	"github.com/jjack/grubstation/internal/config"
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
