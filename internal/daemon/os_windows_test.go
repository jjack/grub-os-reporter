//go:build windows

package daemon

import (
	"os"
	"os/exec"
	"reflect"
	"testing"
)

func TestShutdownSystem_Success(t *testing.T) {
	oldExecCommand := execCommand
	defer func() { execCommand = oldExecCommand }()

	execCommand = func(name string, arg ...string) *exec.Cmd {
		if name != "shutdown" {
			t.Errorf("expected command 'shutdown', got '%s'", name)
		}
		expectedArgs := []string{"/s", "/t", "0"}
		if !reflect.DeepEqual(arg, expectedArgs) {
			t.Errorf("expected args %v, got %v", expectedArgs, arg)
		}
		
		// Return a command that succeeds (like 'cmd /c exit 0')
		return exec.Command("cmd", "/c", "exit 0")
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
		return exec.Command("cmd", "/c", "exit 1")
	}

	err := shutdownSystem()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
