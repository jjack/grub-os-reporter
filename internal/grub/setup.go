package grub

import (
	_ "embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

var (
	ErrInvalidHAURL = errors.New("invalid home assistant url: scheme and host are required")
	ErrNoGrubTool   = errors.New("neither update-grub nor grub2-mkconfig found in PATH")
)

func generateWaitList(seconds int) string {
	if seconds <= 0 {
		return "1"
	}
	var parts []string
	for i := 1; i <= seconds; i++ {
		parts = append(parts, fmt.Sprintf("%d", i))
	}
	return strings.Join(parts, " ")
}

// Setup creates a GRUB remote boot agent script in /etc/grub.d and updates the GRUB config.
func (g *Grub) Setup(opts SetupOptions) error {
	content, err := g.GenerateScript(opts)
	if err != nil {
		return err
	}

	if err := os.WriteFile(g.HassGrubStationPath, []byte(content), 0o755); err != nil {
		return fmt.Errorf("failed to create grub script (are you running as root?): %w", err)
	}

	if path, err := g.LookPath("update-grub"); err == nil {
		out, err := g.Command(path).CombinedOutput()
		if err != nil {
			return fmt.Errorf("update-grub failed: %s", string(out))
		}
		return nil
	}
	if path, err := g.LookPath("grub2-mkconfig"); err == nil {
		out, err := g.Command(path, "-o", "/boot/grub2/grub.cfg").CombinedOutput()
		if err != nil {
			return fmt.Errorf("grub2-mkconfig failed: %s", string(out))
		}
		return nil
	}
	return ErrNoGrubTool
}

// CheckDrift returns true if the installed GRUB script differs from what the current options would generate.
func (g *Grub) CheckDrift(opts SetupOptions) (bool, error) {
	expected, err := g.GenerateScript(opts)
	if err != nil {
		return false, err
	}

	actual, err := os.ReadFile(g.HassGrubStationPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil // It doesn't exist, so it's definitely drifting (not installed)
		}
		return false, fmt.Errorf("failed to read installed grub script: %w", err)
	}

	return string(actual) != expected, nil
}

func (g *Grub) GenerateScript(opts SetupOptions) (string, error) {
	u, err := url.Parse(opts.TargetURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", ErrInvalidHAURL
	}

	tmpl, err := g.GetTemplate("99_grubstation.tmpl")
	if err != nil {
		return "", fmt.Errorf("failed to parse grub template: %w", err)
	}

	data := struct {
		Host            string
		MACAddress      string
		WebhookID       string
		WaitTimeSeconds int
		WaitList        string
	}{
		Host:            u.Host,
		MACAddress:      opts.TargetMAC,
		WebhookID:       opts.AuthToken,
		WaitTimeSeconds: opts.WaitTimeSeconds,
		WaitList:        generateWaitList(opts.WaitTimeSeconds),
	}

	var content strings.Builder
	if err := tmpl.Execute(&content, data); err != nil {
		return "", fmt.Errorf("failed to execute grub template: %w", err)
	}

	return content.String(), nil
}

// SetupWarning returns a message about potential hardware incompatibilities with GRUB networking.
func (g *Grub) SetupWarning() string {
	return "note: the exact GRUB networking configuration applied by this tool may not work perfectly " +
		"for every motherboard due to how finicky UEFI and network firmware can be across different " +
		"hardware vendors. If your system struggles to connect to the network from within GRUB, " +
		"you may need to manually troubleshoot your GRUB network settings."
}

// Uninstall removes the GRUB remote boot agent script and updates the GRUB config.
func (g *Grub) Uninstall() error {
	if err := os.Remove(g.HassGrubStationPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove grub script: %w", err)
	}

	if path, err := g.LookPath("update-grub"); err == nil {
		out, err := g.Command(path).CombinedOutput()
		if err != nil {
			return fmt.Errorf("update-grub failed: %s", string(out))
		}
		return nil
	}
	if path, err := g.LookPath("grub2-mkconfig"); err == nil {
		out, err := g.Command(path, "-o", "/boot/grub2/grub.cfg").CombinedOutput()
		if err != nil {
			return fmt.Errorf("grub2-mkconfig failed: %s", string(out))
		}
		return nil
	}
	return nil
}
