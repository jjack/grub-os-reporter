package homeassistant

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jjack/grubstation/internal/config"
)

func TestClient_UpdateBootOptions(t *testing.T) {
	var receivedPayload UpdatePayload
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-webhook", nil)
	client.HostInfo.NetInterfaces = func() ([]net.Interface, error) { return nil, nil }
	client.HostInfo.GetAddrs = func(iface net.Interface) ([]net.Addr, error) { return nil, nil }

	cfg := &config.Config{
		Host:      config.HostConfig{MAC: "mac", Interface: "eth0"},
		WakeOnLan: config.WakeOnLanConfig{Address: "1.1.1.255", Port: 9},
	}
	state := &config.State{}

	err := client.UpdateBootOptions(cfg, state, []string{"Ubuntu", "Windows"})
	if err != nil {
		t.Fatalf("UpdateBootOptions failed: %v", err)
	}

	if receivedPayload.Action != ActionUpdateAction {
		t.Errorf("expected action %s, got %s", ActionUpdateAction, receivedPayload.Action)
	}
	if len(receivedPayload.BootOptions) != 2 {
		t.Errorf("expected 2 boot options, got %d", len(receivedPayload.BootOptions))
	}
	if receivedPayload.WolBroadcastAddress != "1.1.1.255" {
		t.Errorf("expected wol address 1.1.1.255, got %s", receivedPayload.WolBroadcastAddress)
	}
}

func TestClient_Push_InvalidURL(t *testing.T) {
	client := NewClient(":\x00invalid%url", "test", nil)
	err := client.PostWebhook(UpdatePayload{})
	if err == nil {
		t.Fatal("expected error on invalid URL, got nil")
	}
}

func TestClient_Push_HostError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-webhook", nil)
	err := client.PostWebhook(UpdatePayload{})
	if err == nil {
		t.Fatal("expected error on server 500, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected status code") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// This tests HTTP Client errors in Do() for Push
func TestClient_Push_HttpClientError(t *testing.T) {
	// Create client with invalid base url matching protocol scheme error
	client := NewClient("http://127.0.0.1:0", "test", nil)
	err := client.PostWebhook(UpdatePayload{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClient_Push_NotOKResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ERROR"))
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-webhook", nil)
	err := client.PostWebhook(UpdatePayload{})
	if err == nil || !strings.Contains(err.Error(), "unexpected response from home assistant") {
		t.Fatalf("expected unexpected response error, got %v", err)
	}
}

func TestClient_Push_MarshalError(t *testing.T) {
	client := NewClient("http://ha.local", "test", nil)
	// Channels cannot be marshaled to JSON
	err := client.PostWebhook(make(chan int))
	if err == nil || !strings.Contains(err.Error(), "failed to marshal push payload") {
		t.Fatalf("expected marshal error, got %v", err)
	}
}
