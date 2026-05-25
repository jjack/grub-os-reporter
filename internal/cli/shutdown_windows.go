//go:build windows

package cli

import (
	"fmt"
	"os/exec"
)

func shutdownSystem() error {
	out, err := exec.Command("shutdown", "/s", "/t", "0").CombinedOutput()
	if err != nil {
		return fmt.Errorf("shutdown failed: %w (%s)", err, string(out))
	}
	return nil
}
