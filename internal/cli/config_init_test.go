package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigInitCmd(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grubstation-config-init-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	outputFile := filepath.Join(tempDir, "config.yaml")
	cmd := NewConfigInitCmd()

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--output", outputFile})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "Default configuration generated at") {
		t.Errorf("expected output to contain success message, got %s", out.String())
	}

	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("expected config file to be created at %s", outputFile)
	}

	// Verify it fails if file already exists
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when config file already exists, got nil")
	} else if !strings.Contains(err.Error(), "config file already exists") {
		t.Errorf("expected error message about file already existing, got %v", err)
	}
}
