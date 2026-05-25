//go:build linux

package host

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

func (h *Host) isPhysicalInterface(inf net.Interface) bool {
	virtualInterfaces := []string{"veth", "docker", "br-", "virbr", "vmnet", "vboxnet"}
	for _, prefix := range virtualInterfaces {
		if strings.HasPrefix(inf.Name, prefix) {
			log.Debug().Str("name", inf.Name).Str("prefix", prefix).Msg("Interface is virtual (skipping)")
			return false
		}
	}

	path := fmt.Sprintf("/sys/class/net/%s/device", inf.Name)
	_, err := h.OsStat(path)
	return !os.IsNotExist(err)
}

func Platform() string {
	return "linux"
}

// GetFQDN attempts to resolve the Fully Qualified Domain Name for a given hostname.
func (h *Host) GetFQDN(hostname string, _ *net.Interface) string {
	if cname, err := h.NetLookupCNAME(hostname); err == nil && cname != "" {
		return strings.TrimSuffix(cname, ".")
	}
	return hostname
}
