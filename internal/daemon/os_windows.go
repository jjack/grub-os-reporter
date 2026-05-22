//go:build windows

package daemon

import (
	"os/exec"
)

var execCommand = exec.Command

func shutdownSystem() error {
	return execCommand("shutdown", "/s", "/t", "0").Run()
}
