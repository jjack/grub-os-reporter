//go:build linux

package servicemanager

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestSystemd(t *testing.T) {
	s := NewSystemd().(*Systemd)

	t.Run("Basic", func(t *testing.T) {
		if s.Name() != "systemd" {
			t.Errorf("expected systemd, got %s", s.Name())
		}
	})

	t.Run("IsActive", func(t *testing.T) {
		// This depends on the environment, but we can at least call it
		_ = s.IsActive()
	})

	t.Run("Install_Success", func(t *testing.T) {
		tempDir := t.TempDir()
		s.ServicePath = tempDir + "/grubstation.service"
		s.OsExecutable = func() (string, error) { return "/bin/gs", nil }
		s.OsWriteFile = func(name string, data []byte, perm os.FileMode) error { return nil }
		s.ExecCommand = func(name string, arg ...string) *exec.Cmd {
			return exec.Command("true")
		}

		err := s.Install("cfg.yaml")
		if err != nil {
			t.Fatalf("Install failed: %v", err)
		}
	})

	t.Run("Install_Errors", func(t *testing.T) {
		s.OsExecutable = func() (string, error) { return "", errors.New("exec error") }
		err := s.Install("cfg.yaml")
		if err == nil {
			t.Error("expected error from OsExecutable, got nil")
		}

		s.OsExecutable = func() (string, error) { return "/bin/gs", nil }
		s.OsWriteFile = func(name string, data []byte, perm os.FileMode) error { return errors.New("write error") }
		err = s.Install("cfg.yaml")
		if err == nil || !strings.Contains(err.Error(), "failed to write systemd service file") {
			t.Errorf("expected write error, got %v", err)
		}

		s.OsWriteFile = func(name string, data []byte, perm os.FileMode) error { return nil }
		s.ExecCommand = func(name string, arg ...string) *exec.Cmd {
			return exec.Command("false")
		}
		err = s.Install("cfg.yaml")
		if err == nil || !strings.Contains(err.Error(), "failed to reload systemd daemon") {
			t.Errorf("expected reload error, got %v", err)
		}
	})

	t.Run("Uninstall", func(t *testing.T) {
		s.ExecCommand = func(name string, arg ...string) *exec.Cmd {
			return exec.Command("true")
		}
		s.OsRemove = func(name string) error { return nil }
		err := s.Uninstall()
		if err != nil {
			t.Errorf("Uninstall failed: %v", err)
		}

		s.OsRemove = func(name string) error { return errors.New("remove error") }
		err = s.Uninstall()
		if err == nil {
			t.Error("expected error from OsRemove, got nil")
		}
	})

	t.Run("StartStop", func(t *testing.T) {
		s.ExecCommand = func(name string, arg ...string) *exec.Cmd {
			return exec.Command("true")
		}
		if err := s.Start(); err != nil {
			t.Errorf("Start failed: %v", err)
		}
		if err := s.Stop(); err != nil {
			t.Errorf("Stop failed: %v", err)
		}

		s.ExecCommand = func(name string, arg ...string) *exec.Cmd {
			return exec.Command("false")
		}
		if err := s.Start(); err == nil {
			t.Error("expected Start error, got nil")
		}
		if err := s.Stop(); err == nil {
			t.Error("expected Stop error, got nil")
		}
	})

	t.Run("RegisterDefaultServices", func(t *testing.T) {
		r := NewRegistry()
		RegisterDefaultServices(r)
		if r.Get("systemd") == nil {
			t.Error("systemd not registered")
		}
	})
}

func TestSystemd_IsInstalled(t *testing.T) {
	temp := t.TempDir()
	path := temp + "/service"
	s := &Systemd{ServicePath: path}

	t.Run("Installed", func(t *testing.T) {
		_ = os.WriteFile(path, []byte(""), 0o644)
		installed, err := s.IsInstalled()
		if err != nil {
			t.Fatal(err)
		}
		if !installed {
			t.Error("expected installed=true")
		}
	})

	t.Run("NotInstalled", func(t *testing.T) {
		_ = os.Remove(path)
		installed, err := s.IsInstalled()
		if err != nil {
			t.Fatal(err)
		}
		if installed {
			t.Error("expected installed=false")
		}
	})
}

func TestSystemd_CheckPermissions(t *testing.T) {
	s := &Systemd{}

	t.Run("Root", func(t *testing.T) {
		s.OsGetuid = func() int { return 0 }
		if err := s.CheckPermissions(); err != nil {
			t.Errorf("expected no error for root, got %v", err)
		}
	})

	t.Run("NonRoot", func(t *testing.T) {
		s.OsGetuid = func() int { return 1000 }
		if err := s.CheckPermissions(); err == nil {
			t.Error("expected error for non-root, got nil")
		}
	})
}

func TestSystemd_Install_AbsError(t *testing.T) {
	// This is hard to trigger with filepath.Abs unless we mess with working directory
	temp := t.TempDir()
	_ = os.Chdir(temp)
	_ = os.RemoveAll(temp)

	s := NewSystemd().(*Systemd)
	err := s.Install("cfg.yaml")
	if err == nil {
		t.Error("expected error from filepath.Abs, got nil")
	}
}
