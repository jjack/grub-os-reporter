package homeassistant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/host"
)

const (
	HttpClientTimeout = 10 * time.Second
	OKResponse        = "OK"
)

type Client struct {
	BaseURL    string
	WebhookID  string
	HTTPClient *http.Client
	HostInfo   *host.Host
}

type Action string

const (
	ActionRegisterAction Action = "register_agent_token"
	ActionUpdateAction   Action = "update_boot_options"
	ActionUnregisterHost Action = "unregister_host"
)

type CommonPayload struct {
	Action     Action `json:"action"`
	MACAddress string `json:"mac"`
	Address    string `json:"address"`
}

type RegistrationPayload struct {
	CommonPayload
	AgentToken string `json:"agent_token,omitempty"`
	AgentPort  int    `json:"agent_port,omitempty"`
}

type UpdatePayload struct {
	CommonPayload
	BootOptions         []string `json:"boot_options"`
	WolBroadcastAddress string   `json:"broadcast_address,omitempty"`
	WolBroadcastPort    int      `json:"broadcast_port,omitempty"`
}

func NewClient(baseURL, webhookID string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: HttpClientTimeout}
	}
	return &Client{
		BaseURL:    baseURL,
		WebhookID:  webhookID,
		HTTPClient: httpClient,
		HostInfo:   host.New(),
	}
}

func (c *Client) PostWebhook(payload any) error {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid base url: %w", err)
	}
	targetURL := u.JoinPath("api/webhook", c.WebhookID).String()

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal push payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request to home assistant failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code received from home assistant: %d", resp.StatusCode)
	}

	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if string(bodyBytes) != OKResponse {
		return fmt.Errorf("unexpected response from home assistant (do you have the right webhook_id?): %s", string(bodyBytes))
	}

	return nil
}

func (c *Client) RegisterAgent(cfg *config.Config, state *config.State) error {
	payload := RegistrationPayload{
		CommonPayload: CommonPayload{
			Action:     ActionRegisterAction,
			MACAddress: cfg.Host.MAC,
			Address:    c.HostInfo.GetIPForInterface(cfg.Host.Interface),
		},
		AgentToken: state.APIKey,
		AgentPort:  cfg.Daemon.Port,
	}
	return c.PostWebhook(payload)
}

func (c *Client) UpdateBootOptions(cfg *config.Config, state *config.State, options []string) error {
	payload := UpdatePayload{
		CommonPayload: CommonPayload{
			Action:     ActionUpdateAction,
			MACAddress: cfg.Host.MAC,
			Address:    c.HostInfo.GetIPForInterface(cfg.Host.Interface),
		},
		BootOptions:         options,
		WolBroadcastAddress: cfg.WakeOnLan.Address,
		WolBroadcastPort:    cfg.WakeOnLan.Port,
	}
	return c.PostWebhook(payload)
}

func (c *Client) UnregisterHost(cfg *config.Config, state *config.State) error {
	payload := CommonPayload{
		Action:     ActionUnregisterHost,
		MACAddress: cfg.Host.MAC,
		Address:    c.HostInfo.GetIPForInterface(cfg.Host.Interface),
	}
	return c.PostWebhook(payload)
}
