package config

import (
	"errors"
	log "github.com/rs/zerolog/log"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const WebHookMaxLength = 64

var (
	ErrAddressEmpty               = errors.New("address cannot be empty")
	ErrInvalidWolBroadcastAddress = errors.New("invalid WOL address: must be a valid IP address")
	ErrInvalidWolBroadcastPort    = errors.New("invalid WOL port: must be a number between 1 and 65535")
	ErrGrubConfigPathEmpty        = errors.New("grub config path cannot be empty")
	ErrGrubWaitTimeInvalid        = errors.New("grub wait time must be a number between 1 and 300 seconds")
	ErrGrubWaitTimeEmpty          = errors.New("grub wait time cannot be empty")
	ErrGrubWaitTimeNotNumber      = errors.New("grub wait time must be a number")
	ErrInvalidHost                = errors.New("host must be a valid IP address or hostname (letters, numbers, hyphens, dots)")
	ErrInvalidMACAddress          = errors.New("invalid MAC address format")
	ErrInvalidURL                 = errors.New("invalid URL format")
	ErrMACAddressEmpty            = errors.New("mac address cannot be empty")
	ErrHTTPSUnsupported           = errors.New("https is not supported by grub; please use an http:// url")
	ErrURLEmpty                   = errors.New("url cannot be empty")
	ErrWebhookIDEmpty             = errors.New("webhook id cannot be empty")
	ErrWebhookIDInvalidChar       = errors.New("webhook id can only contain letters, numbers, hyphens, and underscores")
	ErrWebhookIDWrongSize         = errors.New("webhook id should be 64 characters long")
)

func ValidateGrubWaitTime(v string) error {
	if v == "" {
		return ErrGrubWaitTimeEmpty
	}
	val, err := strconv.Atoi(v)
	if err != nil {
		return ErrGrubWaitTimeNotNumber
	}
	if val < 1 || val > 300 {
		return ErrGrubWaitTimeInvalid
	}
	return nil
}

func ValidateMACAddress(v string) error {
	if v == "" {
		return ErrMACAddressEmpty
	}
	_, err := net.ParseMAC(v)
	if err != nil {
		return ErrInvalidMACAddress
	}
	return nil
}

func ValidateURL(v string) error {
	if v == "" {
		return ErrURLEmpty
	}
	u, err := url.ParseRequestURI(v)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ErrInvalidURL
	}
	return nil
}

func ValidateGrubURL(v string) error {
	if err := ValidateURL(v); err != nil {
		return err
	}
	u, _ := url.Parse(v)
	if strings.ToLower(u.Scheme) == "https" {
		return ErrHTTPSUnsupported
	}
	return nil
}

func ValidateWebhookID(webhookID string) error {
	if webhookID == "" {
		return ErrWebhookIDEmpty
	}

	if len(webhookID) != WebHookMaxLength {
		return ErrWebhookIDWrongSize
	}

	if !regexp.MustCompile(`^[a-z0-9_-]+$`).MatchString(webhookID) {
		return ErrWebhookIDInvalidChar
	}

	return nil
}

func ValidateWolBroadcastAddress(v string) error {
	// empty means use the default address - 255.255.255.255
	if v == "" {
		return nil
	}
	if net.ParseIP(v) == nil {
		return ErrInvalidWolBroadcastAddress
	}
	return nil
}

func ValidatePort(v string) error {
	// empty means use the default port - 9
	if v == "" {
		return nil
	}
	port, err := strconv.Atoi(v)
	if err != nil || port < 1 || port > 65535 {
		log.Error().Interface("port", port).Msg("Invalid WOL port")
		return ErrInvalidWolBroadcastPort
	}
	return nil
}

func (c *Config) Validate() error {
	if err := ValidateMACAddress(c.Host.MACAddress); err != nil {
		return err
	}
	if err := ValidateHost(c.Host.Address); err != nil {
		return err
	}
	if err := ValidateURL(c.HomeAssistant.URL); err != nil {
		return err
	}
	if err := ValidateWebhookID(c.HomeAssistant.WebhookID); err != nil {
		return err
	}

	if c.Daemon.ReportBootOptions {
		if c.Grub == nil || c.Grub.ConfigPath == "" {
			return ErrGrubConfigPathEmpty
		}
		if c.Grub.URL != "" {
			if err := ValidateGrubURL(c.Grub.URL); err != nil {
				return err
			}
		}
	}

	if c.WakeOnLan != nil {
		portStr := ""
		if c.WakeOnLan.Port != 0 {
			portStr = strconv.Itoa(c.WakeOnLan.Port)
		}
		if err := ValidatePort(portStr); err != nil {
			return err
		}
		if err := ValidateWolBroadcastAddress(c.WakeOnLan.Address); err != nil {
			return err
		}
	}
	return nil
}

// hosts can be IP addresses or hostnames, but must not be empty and must only contain valid characters
func ValidateHost(v string) error {
	if v == "" {
		return ErrAddressEmpty
	}

	// it's a valid ip
	if net.ParseIP(v) != nil {
		return nil
	}

	// it's a valid hostname
	if regexp.MustCompile(`^[a-zA-Z0-9-.]+$`).MatchString(v) {
		return nil
	}
	return ErrInvalidHost
}
