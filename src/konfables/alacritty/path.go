package alacritty

import (
	"os"
	"path/filepath"

	"github.com/getkonfi/konfi/pkg"
)

// DefaultConfigPath returns the Alacritty config file konfi should edit.
func DefaultConfigPath() string {
	candidates := alacrittyConfigCandidates()
	for _, path := range candidates {
		if pkg.FileExists(path) {
			return path
		}
	}
	return candidates[0]
}

func alacrittyConfigCandidates() []string {
	home, _ := os.UserHomeDir()
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		xdg = filepath.Join(home, ".config")
	}
	return []string{
		filepath.Join(xdg, "alacritty", "alacritty.toml"),
		filepath.Join(xdg, "alacritty.toml"),
		filepath.Join(home, ".config", "alacritty", "alacritty.toml"),
		filepath.Join(home, ".alacritty.toml"),
	}
}
