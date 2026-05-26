package homeassistant

import (
	"net"
	"testing"
	"time"

	"github.com/grandcat/zeroconf"
)

func TestExtractURLs(t *testing.T) {
	tests := []struct {
		name     string
		entry    *zeroconf.ServiceEntry
		expected []string
	}{
		{
			name: "internal_url and ip present",
			entry: &zeroconf.ServiceEntry{
				ServiceRecord: zeroconf.ServiceRecord{
					Instance: "Home",
				},
				AddrIPv4: []net.IP{net.ParseIP("192.168.1.100")},
				Port:     8123,
				Text: []string{
					"internal_url=http://ha.local:8123",
					"base_url=http://base.local",
				},
			},
			expected: []string{"http://ha.local:8123", "http://base.local", "http://192.168.1.100:8123"},
		},
		{
			name: "only ip present",
			entry: &zeroconf.ServiceEntry{
				ServiceRecord: zeroconf.ServiceRecord{
					Instance: "Home",
				},
				AddrIPv4: []net.IP{net.ParseIP("192.168.1.100")},
				Port:     8123,
			},
			expected: []string{"http://192.168.1.100:8123"},
		},
		{
			name: "no useful info",
			entry: &zeroconf.ServiceEntry{
				ServiceRecord: zeroconf.ServiceRecord{
					Instance: "Home",
				},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls := extractURLs(tt.entry)
			if len(urls) != len(tt.expected) {
				t.Fatalf("expected %d urls, got %d: %v", len(tt.expected), len(urls), urls)
			}
			for i := range urls {
				if urls[i] != tt.expected[i] {
					t.Errorf("at index %d: expected %s, got %s", i, tt.expected[i], urls[i])
				}
			}
		})
	}
}

func TestDiscover_Timeout(t *testing.T) {
	// Set a very short timeout for the test
	oldTimeout := discoveryTimeout
	discoveryTimeout = 10 * time.Millisecond
	defer func() { discoveryTimeout = oldTimeout }()

	// This might still return results if there's a real HA on the network,
	// but it shouldn't fail.
	_, _ = Discover()
}
