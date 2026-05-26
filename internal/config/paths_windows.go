//go:build windows

package config

import (
	"os"
	"path/filepath"
)

func DefaultConfigPath() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = "C:\\ProgramData"
	}
	return filepath.Join(programData, "GrubStation", "config.yaml")
}

func DefaultStatePath() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = "C:\\ProgramData"
	}
	return filepath.Join(programData, "GrubStation", "state.json")
}
