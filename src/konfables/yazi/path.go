package yazi

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/getkonfi/konfi/pkg"
)

func DefaultConfigPath() string {
	if dir := strings.TrimSpace(os.Getenv("YAZI_CONFIG_HOME")); dir != "" {
		return filepath.Join(dir, "yazi.toml")
	}
	return pkg.XDGConfigPath("yazi", "yazi.toml")
}
