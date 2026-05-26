package grub

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGrub_Discover(t *testing.T) {
	tempDir := t.TempDir()
	fakeGrubPath := filepath.Join(tempDir, "grub.cfg")
	if err := os.WriteFile(fakeGrubPath, []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write temp grub config: %v", err)
	}

	originalPaths := knownConfigPaths
	defer func() { knownConfigPaths = originalPaths }()

	// Test success cases
	knownConfigPaths = []string{fakeGrubPath}

	g := &Grub{}
	path, err := g.DiscoverConfigPath()
	if err != nil {
		t.Errorf("expected no error from DiscoverConfigPath, got %v", err)
	}
	if path != fakeGrubPath {
		t.Errorf("expected discovered path %s, got %s", fakeGrubPath, path)
	}

	// Test error case
	knownConfigPaths = []string{"/nonexistent/grub.cfg"}
	_, err = g.DiscoverConfigPath()
	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("expected ErrConfigNotFound, got %v", err)
	}
}

func TestGrub_AutoDiscovery(t *testing.T) {
	// Temporarily override the tracked paths to point to a temp dir so that the environment doesn't affect it
	tempDir := t.TempDir()
	fakeGrubPath := filepath.Join(tempDir, "grub.cfg")
	if err := os.WriteFile(fakeGrubPath, []byte("menuentry 'Arch Linux' { }"), 0o644); err != nil {
		t.Fatalf("failed to write temp grub config: %v", err)
	}

	originalPaths := knownConfigPaths
	defer func() { knownConfigPaths = originalPaths }()
	knownConfigPaths = []string{fakeGrubPath}

	g := &Grub{}
	bootOptions, err := g.GetBootOptions()
	if err != nil {
		t.Fatalf("expected auto-discovery to find grub config without error, got: %v", err)
	}

	if len(bootOptions) != 1 || bootOptions[0] != "Arch Linux" {
		t.Errorf("expected 'Arch Linux' from auto-discovered file, got %v", bootOptions)
	}
}

func TestGrub_AutoDiscovery_Fail(t *testing.T) {
	originalPaths := knownConfigPaths
	defer func() { knownConfigPaths = originalPaths }()
	knownConfigPaths = []string{"/tmp/definitely-do-not-exist"}

	g := &Grub{}
	_, err := g.GetBootOptions()
	if err == nil {
		t.Fatal("expected failure to find any grub config")
	}
}
