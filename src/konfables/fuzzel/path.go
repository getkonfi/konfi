package fuzzel

import "github.com/eminert/konfi/pkg"

func DefaultConfigPath() string {
	return pkg.XDGConfigPath("fuzzel", "fuzzel.ini")
}
