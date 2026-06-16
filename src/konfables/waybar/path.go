package waybar

import "github.com/getkonfi/konfi/pkg"

func DefaultConfigPath() string {
	configPath := pkg.XDGConfigPath("waybar", "config")
	if pkg.FileExists(configPath) {
		return configPath
	}
	jsoncPath := pkg.XDGConfigPath("waybar", "config.jsonc")
	if pkg.FileExists(jsoncPath) {
		return jsoncPath
	}
	return configPath
}
