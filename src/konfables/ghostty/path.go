package ghostty

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/eminert/konfi/pkg"
)

// DefaultConfigPath returns the Ghostty config file konfi should edit.
func DefaultConfigPath() string {
	return defaultConfigPath(runtime.GOOS)
}

func defaultConfigPath(goos string) string {
	xdgConfig := ghosttyXDGConfigPath("config.ghostty")
	defaultPath := xdgConfig
	if goos == "darwin" && os.Getenv("XDG_CONFIG_HOME") == "" {
		defaultPath = ghosttyMacConfigPath("config.ghostty")
	}

	candidates := []string{
		xdgConfig,
		ghosttyXDGConfigPath("config"),
	}
	if goos == "darwin" {
		candidates = append(candidates,
			ghosttyMacConfigPath("config.ghostty"),
			ghosttyMacConfigPath("config"),
		)
	}

	for i := len(candidates) - 1; i >= 0; i-- {
		if pkg.FileExists(candidates[i]) {
			return candidates[i]
		}
	}
	return defaultPath
}

func ghosttyXDGConfigPath(file string) string {
	return pkg.XDGConfigPath("ghostty", file)
}

func ghosttyMacConfigPath(file string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "com.mitchellh.ghostty", file)
}
