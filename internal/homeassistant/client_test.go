package homeassistant

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jjack/grubstation/internal/config"
)

func TestClient_Push(t *testing.T) {
	var receivedPayload RegistrationPayload
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "api/webhook/test-webhook") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}

		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-webhook", nil)
	payload := RegistrationPayload{
		CommonPayload: CommonPayload{
			Action:     ActionRegisterAction,
			MACAddress: "aa:bb:cc:dd",
			Address:    "10.0.0.1",
		},
	}

	err := client.PostWebhook(context.Background(), payload)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if receivedPayload.Action != ActionRegisterAction {
		t.Errorf("expected action %s, got %s", ActionRegisterAction, receivedPayload.Action)
	}
	if receivedPayload.MACAddress != "aa:bb:cc:dd" {
		t.Errorf("expected MAC aa:bb:cc:dd, got %s", receivedPayload.MACAddress)
	}
}

func TestClient_RegisterAgent(t *testing.T) {
	var receivedPayload RegistrationPayload
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-webhook", nil)
	// Mock host info to avoid real network lookup
	client.HostInfo.NetInterfaces = func() ([]net.Interface, error) { return nil, nil }
	client.HostInfo.GetAddrs = func(iface net.Interface) ([]net.Addr, error) { return nil, nil }

	cfg := &config.Config{
		Host:   config.HostConfig{MAC: "mac", Interface: "eth0"},
		Daemon: config.DaemonConfig{Port: 8081},
	}
	state := &config.State{APIKey: "token"}

	err := client.RegisterAgent(context.Background(), cfg, state)
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	if receivedPayload.Action != ActionRegisterAction {
		t.Errorf("expected action %s, got %s", ActionRegisterAction, receivedPayload.Action)
	}
	if receivedPayload.AgentToken != "token" {
		t.Errorf("expected token token, got %s", receivedPayload.AgentToken)
	}
}

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

	err := client.UpdateBootOptions(context.Background(), cfg, state, []string{"Ubuntu", "Windows"})
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

func TestClient_UnregisterHost_Method(t *testing.T) {
	var receivedPayload CommonPayload
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
		Host: config.HostConfig{MAC: "mac", Interface: "eth0"},
	}
	state := &config.State{}

	err := client.UnregisterHost(context.Background(), cfg, state)
	if err != nil {
		t.Fatalf("UnregisterHost failed: %v", err)
	}

	if receivedPayload.Action != ActionUnregisterHost {
		t.Errorf("expected action %s, got %s", ActionUnregisterHost, receivedPayload.Action)
	}
}

func TestClient_Push_InvalidURL(t *testing.T) {
	client := NewClient(":\x00invalid%url", "test", nil)
	err := client.PostWebhook(context.Background(), RegistrationPayload{})
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
	err := client.PostWebhook(context.Background(), RegistrationPayload{})
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
	err := client.PostWebhook(context.Background(), RegistrationPayload{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClient_Push_CreateRequestError(t *testing.T) {
	client := NewClient("http://homeassistant.local:8123", "test", nil)
	// Passing a nil context causes http.NewRequestWithContext to reliably return an error
	//nolint:staticcheck // SA1012: we intentionally pass nil for testing
	err := client.PostWebhook(nil, RegistrationPayload{})
	if err == nil {
		t.Fatal("expected error on nil context, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create http request") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestClient_Push_NotOKResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ERROR"))
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-webhook", nil)
	err := client.PostWebhook(context.Background(), RegistrationPayload{})
	if err == nil || !strings.Contains(err.Error(), "unexpected response from home assistant") {
		t.Fatalf("expected unexpected response error, got %v", err)
	}
}

func TestClient_Push_MarshalError(t *testing.T) {
	client := NewClient("http://ha.local", "test", nil)
	// Channels cannot be marshaled to JSON
	err := client.PostWebhook(context.Background(), make(chan int))
	if err == nil || !strings.Contains(err.Error(), "failed to marshal push payload") {
		t.Fatalf("expected marshal error, got %v", err)
	}
}
