package grub

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

// fakeExecCommand wrappers route the exec call back to the test binary's TestHelperProcess
func fakeExecCommandSuccess(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	return cmd
}

func fakeExecCommandFail(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", "fail"}
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	return cmd
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) > 0 && args[0] == "fail" {
		os.Exit(1)
	}
	os.Exit(0)
}

func TestGrub_Setup_Success(t *testing.T) {
	tempDir := t.TempDir()
	fakeScriptPath := filepath.Join(tempDir, "99_ha_grub_os_reporter")

	g := NewGrub()
	g.HassGrubStationPath = fakeScriptPath
	g.Command = fakeExecCommandSuccess

	// Test success using update-grub
	g.LookPath = func(file string) (string, error) {
		if file == "update-grub" {
			return "/fake/update-grub", nil
		}
		return "", errors.New("not found")
	}

	err := g.Setup(SetupOptions{
		TargetMAC:       "aa:bb:cc:dd:ee:ff",
		TargetURL:       "http://hass.local:8123",
		AuthToken:       "test_webhook",
		WaitTimeSeconds: 2,
	})
	if err != nil {
		t.Fatalf("expected successful install, got %v", err)
	}

	content, _ := os.ReadFile(fakeScriptPath)
	if !strings.Contains(string(content), "http,hass.local:8123") || !strings.Contains(string(content), "aa:bb:cc:dd:ee:ff") || !strings.Contains(string(content), "test_webhook") {
		t.Errorf("template not rendered correctly: %s", string(content))
	}

	// Test fallback success using grub2-mkconfig
	g.LookPath = func(file string) (string, error) {
		if file == "grub2-mkconfig" {
			return "/fake/grub2-mkconfig", nil
		}
		return "", errors.New("not found")
	}

	err = g.Setup(SetupOptions{
		TargetMAC:       "aa:bb:cc:dd:ee:ff",
		TargetURL:       "http://hass.local:8123",
		AuthToken:       "test_webhook",
		WaitTimeSeconds: 2,
	})
	if err != nil {
		t.Fatalf("expected successful install with grub2-mkconfig, got %v", err)
	}
}

func TestGrub_Setup_Errors(t *testing.T) {
	g := NewGrub()

	// 1. Invalid URL
	err := g.Setup(SetupOptions{TargetMAC: "mac", TargetURL: "://bad-url", AuthToken: "test_webhook", WaitTimeSeconds: 2})
	if !errors.Is(err, ErrInvalidHAURL) {
		t.Fatalf("expected ErrInvalidHAURL, got %v", err)
	}

	// 2. File creation failure
	g.HassGrubStationPath = "/this/path/does/not/exist/99_script"
	err = g.Setup(SetupOptions{TargetMAC: "mac", TargetURL: "http://hass.local", AuthToken: "test_webhook", WaitTimeSeconds: 2})
	if err == nil || !strings.Contains(err.Error(), "failed to create grub script") {
		t.Fatalf("expected file creation error, got %v", err)
	}

	// Fix path for subsequent tests
	tempDir := t.TempDir()
	g.HassGrubStationPath = filepath.Join(tempDir, "99_ha_grub_os_reporter")

	// 3. No binary found in PATH
	g.LookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}
	err = g.Setup(SetupOptions{TargetMAC: "mac", TargetURL: "http://hass.local", AuthToken: "test_webhook", WaitTimeSeconds: 2})
	if !errors.Is(err, ErrNoGrubTool) {
		t.Fatalf("expected ErrNoGrubTool, got %v", err)
	}

	// 4. update-grub command execution fails
	g.LookPath = func(file string) (string, error) {
		if file == "update-grub" {
			return "/fake/update-grub", nil
		}
		return "", errors.New("not found")
	}
	g.Command = fakeExecCommandFail
	err = g.Setup(SetupOptions{TargetMAC: "mac", TargetURL: "http://hass.local", AuthToken: "test_webhook", WaitTimeSeconds: 2})
	if err == nil || !strings.Contains(err.Error(), "update-grub failed") {
		t.Fatalf("expected update-grub execution error, got %v", err)
	}

	// 5. grub2-mkconfig command execution fails
	g.LookPath = func(file string) (string, error) {
		if file == "grub2-mkconfig" {
			return "/fake/grub2-mkconfig", nil
		}
		return "", errors.New("not found")
	}
	err = g.Setup(SetupOptions{TargetMAC: "mac", TargetURL: "http://hass.local", AuthToken: "test_webhook", WaitTimeSeconds: 2})
	if err == nil || !strings.Contains(err.Error(), "grub2-mkconfig failed") {
		t.Fatalf("expected grub2-mkconfig execution error, got %v", err)
	}
}

func TestGrub_Setup_TemplateErrors(t *testing.T) {
	g := NewGrub()

	// 1. Template parse error
	g.GetTemplate = func(name string) (*template.Template, error) {
		return nil, errors.New("failed to parse grub template")
	}
	err := g.Setup(SetupOptions{TargetMAC: "mac", TargetURL: "http://hass.local", AuthToken: "test_webhook", WaitTimeSeconds: 2})
	if err == nil || !strings.Contains(err.Error(), "failed to parse grub template") {
		t.Fatalf("expected template parse error, got %v", err)
	}

	// 2. Template execute error
	g.GetTemplate = func(name string) (*template.Template, error) {
		return template.New(name).Parse("{{ .Host.NonExistentField }}")
	}
	err = g.Setup(SetupOptions{TargetMAC: "mac", TargetURL: "http://hass.local", AuthToken: "test_webhook", WaitTimeSeconds: 2})
	if err == nil || !strings.Contains(err.Error(), "failed to execute grub template") {
		t.Fatalf("expected template execute error, got %v", err)
	}
}

func TestGrub_SetupWarning(t *testing.T) {
	g := &Grub{}
	warning := g.SetupWarning()
	if !strings.Contains(warning, "troubleshoot your GRUB network settings") {
		t.Errorf("expected warning to mention troubleshooting, got: %s", warning)
	}
}

func TestGenerateWaitList(t *testing.T) {
	if got := generateWaitList(0); got != "1" {
		t.Errorf("expected 1, got %q", got)
	}
	if got := generateWaitList(-1); got != "1" {
		t.Errorf("expected 1, got %q", got)
	}
	if got := generateWaitList(3); got != "1 2 3" {
		t.Errorf("expected '1 2 3', got %q", got)
	}
}

func TestGrub_Uninstall(t *testing.T) {
	tempDir := t.TempDir()

	g := NewGrub()
	g.HassGrubStationPath = tempDir + "/99_grubstation"

	// Pre-create file
	_ = os.WriteFile(g.HassGrubStationPath, []byte(""), 0o755)

	g.LookPath = func(file string) (string, error) {
		if file == "update-grub" {
			return "/bin/true", nil
		}
		return "", errors.New("not found")
	}
	g.Command = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	err := g.Uninstall()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(g.HassGrubStationPath); !os.IsNotExist(err) {
		t.Error("expected grub script to be removed")
	}
}

func TestGrub_Uninstall_NoFile(t *testing.T) {
	tempDir := t.TempDir()

	g := NewGrub()
	g.HassGrubStationPath = tempDir + "/non-existent"
	g.LookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}

	err := g.Uninstall()
	if err != nil {
		t.Fatalf("expected no error when file is already gone, got %v", err)
	}
}

func TestGrub_Uninstall_RemoveError(t *testing.T) {
	// Use a non-empty directory to cause remove error
	tempDir := t.TempDir()

	g := NewGrub()
	g.HassGrubStationPath = tempDir
	_ = os.WriteFile(filepath.Join(tempDir, "keep"), []byte(""), 0o644)

	err := g.Uninstall()
	if err == nil {
		t.Fatal("expected error when removing a non-empty directory, got nil")
	}
}

func TestGrub_Uninstall_Grub2Mkconfig(t *testing.T) {
	tempDir := t.TempDir()

	g := NewGrub()
	g.HassGrubStationPath = tempDir + "/99_grubstation"

	g.LookPath = func(file string) (string, error) {
		if file == "grub2-mkconfig" {
			return "/bin/true", nil
		}
		return "", errors.New("not found")
	}
	g.Command = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	err := g.Uninstall()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGrub_Uninstall_UpdateGrubError(t *testing.T) {
	tempDir := t.TempDir()
	fakeScriptPath := filepath.Join(tempDir, "99_grubstation")

	g := NewGrub()
	g.HassGrubStationPath = fakeScriptPath
	g.LookPath = func(file string) (string, error) {
		if file == "update-grub" {
			return "/bin/false", nil
		}
		return "", errors.New("not found")
	}
	g.Command = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	err := g.Uninstall()
	if err == nil || !strings.Contains(err.Error(), "update-grub failed") {
		t.Errorf("expected update-grub failure, got %v", err)
	}
}

func TestGrub_CheckDrift(t *testing.T) {
	tempDir := t.TempDir()
	fakeScriptPath := filepath.Join(tempDir, "99_grubstation")

	g := NewGrub()
	g.HassGrubStationPath = fakeScriptPath

	opts := SetupOptions{
		TargetMAC:       "aa:bb:cc:dd:ee:ff",
		TargetURL:       "http://hass.local:8123",
		AuthToken:       "test_webhook",
		WaitTimeSeconds: 2,
	}

	// 1. File does not exist (should drift)
	drift, err := g.CheckDrift(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !drift {
		t.Error("expected drift when file does not exist")
	}

	// 2. File matches (no drift)
	content, err := g.GenerateScript(opts)
	if err != nil {
		t.Fatalf("failed to generate script: %v", err)
	}
	if err := os.WriteFile(fakeScriptPath, []byte(content), 0o755); err != nil {
		t.Fatalf("failed to write fake script: %v", err)
	}

	drift, err = g.CheckDrift(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if drift {
		t.Error("expected no drift when file matches")
	}

	// 3. File differs (should drift)
	optsDifferent := opts
	optsDifferent.WaitTimeSeconds = 10
	drift, err = g.CheckDrift(optsDifferent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !drift {
		t.Error("expected drift when config differs")
	}

	// 4. Read error (other than NotExist)
	g.HassGrubStationPath = filepath.Join(tempDir, "cannot_read_dir")
	if err := os.Mkdir(g.HassGrubStationPath, 0o000); err != nil {
		t.Fatalf("failed to create unreadable directory: %v", err)
	}
	defer func() { _ = os.Chmod(g.HassGrubStationPath, 0o755) }() // Clean up

	_, err = g.CheckDrift(opts)
	if err == nil {
		t.Error("expected error when reading file fails")
	}
}

func TestGrub_GenerateScript_Errors(t *testing.T) {
	g := NewGrub()

	// Invalid URL
	_, err := g.GenerateScript(SetupOptions{TargetURL: "://bad"})
	if !errors.Is(err, ErrInvalidHAURL) {
		t.Fatalf("expected ErrInvalidHAURL, got %v", err)
	}

	// Template error
	g.GetTemplate = func(name string) (*template.Template, error) {
		return nil, errors.New("failed to parse grub template")
	}
	_, err = g.GenerateScript(SetupOptions{TargetURL: "http://hass.local"})
	if err == nil || !strings.Contains(err.Error(), "failed to parse grub template") {
		t.Fatalf("expected template parse error, got %v", err)
	}
}
