package cst

import (
	"os"
	"path/filepath"
)

const (
	AppName    = "konfi"
	AppVersion = "0.1.0"

	ConfigFileName = "config.yaml"
	LogFileName    = "konfi.log"
)

// ConfigDir returns ~/.config/konfi.
func ConfigDir() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, AppName)
}

// ConfigFilePath returns the full path to konfi's own config.
func ConfigFilePath() string {
	return filepath.Join(ConfigDir(), ConfigFileName)
}

// LogFilePath returns the full path to the log file.
func LogFilePath() string {
	return filepath.Join(ConfigDir(), LogFileName)
}
