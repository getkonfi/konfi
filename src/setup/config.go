package setup

import (
	"context"
	"os"

	"github.com/emin/konfigurator/setup/cst"

	"gopkg.in/yaml.v3"
)

// KonfConfig holds konfigurator's own preferences.
type KonfConfig struct {
	Theme          string   `yaml:"theme"`
	LogLevel       string   `yaml:"log_level"`
	BrowseLoadsApp bool     `yaml:"browse_loads_app"`
	NerdFont       bool     `yaml:"nerd_font"`
	Bookmarks      []string `yaml:"bookmarks,omitempty"`
}

func defaultConfig() *KonfConfig {
	return &KonfConfig{
		Theme:    "catppuccin",
		LogLevel: "info",
		NerdFont: true,
	}
}

// InitConfig loads ~/.config/konfigurator/config.yaml or creates defaults.
func InitConfig(_ context.Context, app *App) error {
	path := cst.ConfigFilePath()
	cfg := defaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			app.Config = cfg
			return nil
		}
		return err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return err
	}

	app.Config = cfg
	return nil
}

// SaveConfig persists the current config to disk.
func SaveConfig(cfg *KonfConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	path := cst.ConfigFilePath()
	if err := os.MkdirAll(cst.ConfigDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
