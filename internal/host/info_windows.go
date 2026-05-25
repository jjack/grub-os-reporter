//go:build windows

package host

import (
	"fmt"
	"net"
	"strings"
	"unsafe"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
)

func (h *Host) isPhysicalInterface(inf net.Interface) bool {
	var row windows.MibIfRow2
	row.InterfaceIndex = uint32(inf.Index)

	// GetIfEntry2Ex is used as a replacement for GetIfEntry2.
	// Level 0 corresponds to MibIfEntryNormal.
	if err := windows.GetIfEntry2Ex(0, &row); err != nil {
		log.Debug().Interface("name", inf.Name).Err(err).Msg("Failed to get interface entry. Assuming it is physical.")
		return true // Fallback to true if we can't determine
	}

	// ConnectorPresent is the first bit of InterfaceAndOperStatusFlags (bit 0)
	// It is TRUE (1) if the interface has a physical connector.
	return (row.InterfaceAndOperStatusFlags & 0x01) != 0
}

func Platform() string {
	return "windows"
}

func (h *Host) getAdapterDNSSuffix(ifIndex uint32) string {
	// GetAdaptersAddresses is used to retrieve DNS suffixes.
	// We use GAA_FLAG_SKIP_ANYCAST | GAA_FLAG_SKIP_MULTICAST | GAA_FLAG_SKIP_FRIENDLY_NAME
	flags := uint32(windows.GAA_FLAG_SKIP_ANYCAST | windows.GAA_FLAG_SKIP_MULTICAST | windows.GAA_FLAG_SKIP_FRIENDLY_NAME)

	// Initial buffer size
	size := uint32(16384)
	for {
		b := make([]byte, size)
		err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, flags, 0, (*windows.IpAdapterAddresses)(unsafe.Pointer(&b[0])), &size)
		if err == nil {
			for addr := (*windows.IpAdapterAddresses)(unsafe.Pointer(&b[0])); addr != nil; addr = addr.Next {
				if addr.IfIndex == ifIndex || addr.Ipv6IfIndex == ifIndex {
					return windows.UTF16PtrToString(addr.DnsSuffix)
				}
			}
			return ""
		}
		if err != windows.ERROR_BUFFER_OVERFLOW {
			break
		}
		// size is updated by GetAdaptersAddresses, loop again with larger buffer
	}

	return ""
}

// GetFQDN attempts to resolve the Fully Qualified Domain Name for a given hostname.
// On Windows, it tries to append the Connection-specific DNS Suffix of the provided interface.
func (h *Host) GetFQDN(hostname string, inf *net.Interface) string {
	if inf != nil {
		if suffix := h.getAdapterDNSSuffix(uint32(inf.Index)); suffix != "" {
			return fmt.Sprintf("%s.%s", hostname, suffix)
		}
	}

	if cname, err := h.NetLookupCNAME(hostname); err == nil && cname != "" {
		return strings.TrimSuffix(cname, ".")
	}
	return hostname
}
