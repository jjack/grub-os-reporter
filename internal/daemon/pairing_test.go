package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestDaemon_PairUnpair(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port := getFreePort(t)
	d := New(Config{Port: port}, Metadata{}, nil, nil)

	var pairedReq PairRequest
	pairCalled := make(chan bool, 1)
	d.OnPair = func(req PairRequest) error {
		pairedReq = req
		pairCalled <- true
		return nil
	}

	unpairCalled := make(chan bool, 1)
	d.OnUnpair = func() error {
		unpairCalled <- true
		return nil
	}

	done := make(chan error, 1)
	go func() { done <- d.run(ctx) }()

	if err := waitForServer(port); err != nil {
		cancel()
		t.Fatal(err)
	}

	// 1. Pair
	pairPayload := PairRequest{
		WebhookID:   "webhook_id",
		APIKey:      "new_api_key",
		HADaemonURL: "http://ha:8123",
		HAGrubURL:   "http://ha:8081",
	}
	body, _ := json.Marshal(pairPayload)
	resp, err := getTestClient().Post(fmt.Sprintf("http://localhost:%d/pair", port), "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("pair request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	select {
	case <-pairCalled:
		if pairedReq.WebhookID != pairPayload.WebhookID || pairedReq.APIKey != pairPayload.APIKey {
			t.Errorf("OnPair called with wrong data: %+v", pairedReq)
		}
	case <-time.After(time.Second):
		t.Error("OnPair callback not called")
	}

	if d.getAPIKey() != pairPayload.APIKey {
		t.Errorf("expected API key to be updated to %s, got %s", pairPayload.APIKey, d.getAPIKey())
	}

	// 2. Unpair (Authorized)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%d/unpair", port), nil)
	req.Header.Set("Authorization", "Bearer "+pairPayload.APIKey)
	resp, err = getTestClient().Do(req)
	if err != nil {
		t.Fatalf("unpair request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	select {
	case <-unpairCalled:
		// success
	case <-time.After(time.Second):
		t.Error("OnUnpair callback not called")
	}

	if d.getAPIKey() != "" {
		t.Errorf("expected API key to be cleared, got %s", d.getAPIKey())
	}

	// 3. Unpair (Unauthorized)
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%d/unpair", port), nil)
	req.Header.Set("Authorization", "Bearer wrong")
	resp, err = getTestClient().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}

	cancel()
	<-done
}
