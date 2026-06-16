package fuzzel

import "github.com/getkonfi/konfi/pkg"

func DefaultConfigPath() string {
	return pkg.XDGConfigPath("fuzzel", "fuzzel.ini")
}
