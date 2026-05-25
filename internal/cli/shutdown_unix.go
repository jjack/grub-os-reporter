//go:build !windows

package cli

import (
	"fmt"
	"os/exec"
)

func shutdownSystem() error {
	out, err := exec.Command("poweroff").CombinedOutput()
	if err != nil {
		return fmt.Errorf("shutdown failed: %w (%s)", err, string(out))
	}
	return nil
}
