//go:build !windows

package config

func DefaultConfigPath() string {
	return "/etc/grubstation/config.yaml"
}

func DefaultStatePath() string {
	return "/etc/grubstation/state.json"
}
