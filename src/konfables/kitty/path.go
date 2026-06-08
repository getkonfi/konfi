package kitty

import (
	"os"
	"path/filepath"

	"github.com/eminert/konfi/pkg"
)

// DefaultConfigPath returns the kitty config file konfi should edit.
func DefaultConfigPath() string {
	if dir := os.Getenv("KITTY_CONFIG_DIRECTORY"); dir != "" {
		return filepath.Join(dir, "kitty.conf")
	}
	return pkg.XDGConfigPath("kitty", "kitty.conf")
}
