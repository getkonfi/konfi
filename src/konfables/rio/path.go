package rio

import (
	"os"
	"path/filepath"

	"github.com/getkonfi/konfi/pkg"
)

// DefaultConfigPath returns the Rio config file konfi should edit.
func DefaultConfigPath() string {
	if dir := os.Getenv("RIO_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "config.toml")
	}
	return pkg.XDGConfigPath("rio", "config.toml")
}
