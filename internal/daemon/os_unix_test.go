//go:build !windows

package daemon

import (
	"os"
	"os/exec"
	"testing"
)

func TestShutdownSystem_Success(t *testing.T) {
	oldExecCommand := execCommand
	defer func() { execCommand = oldExecCommand }()

	execCommand = func(name string, arg ...string) *exec.Cmd {
		if name != "poweroff" {
			t.Errorf("expected command 'poweroff', got '%s'", name)
		}
		// Return a command that succeeds (like 'true' or 'echo')
		return exec.Command("true")
	}

	err := shutdownSystem()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestShutdownSystem_Error(t *testing.T) {
	oldExecCommand := execCommand
	defer func() { execCommand = oldExecCommand }()

	execCommand = func(name string, arg ...string) *exec.Cmd {
		// Return a command that fails
		return exec.Command("false")
	}

	err := shutdownSystem()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// Helper for more robust mocking if needed
func TestShutdownSystem_HelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		os.Exit(0)
		return
	}

	oldExecCommand := execCommand
	defer func() { execCommand = oldExecCommand }()

	execCommand = func(name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestShutdownSystem_HelperProcess", "--"}
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		return cmd
	}

	err := shutdownSystem()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
