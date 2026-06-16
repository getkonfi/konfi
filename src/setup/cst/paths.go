package cst

import (
	"os"
	"path/filepath"
)

const (
	AppName = "konfi"

	ConfigFileName = "config.yaml"
	LogFileName    = "konfi.log"
)

var AppVersion = "0.1.0"

// ConfigDir returns konfi's platform config directory.
func ConfigDir() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		home, _ := os.UserHomeDir()
		if home != "" {
			base = filepath.Join(home, ".config")
		} else {
			base = "."
		}
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
