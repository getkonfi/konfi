package setup

import (
	"context"
	"os"

	"github.com/eminert/konfi/setup/cst"

	"gopkg.in/yaml.v3"
)

// KonfConfig holds konfigurator's own preferences.
type KonfConfig struct {
	Theme          string            `yaml:"theme"`
	LogLevel       string            `yaml:"log_level"`
	BrowseLoadsApp bool              `yaml:"browse_loads_app"`
	NerdFont       bool              `yaml:"nerd_font"`
	Bookmarks      []string          `yaml:"bookmarks,omitempty"`
	Upstream       *UpstreamSettings `yaml:"upstream,omitempty"`
}

// UpstreamSettings controls the upstream-check maintainer tool.
// optional: unused by the tui, consumed by tools/upstreamcheck.
type UpstreamSettings struct {
	GitHub *GitHubSettings `yaml:"github,omitempty"`
	GitLab *GitLabSettings `yaml:"gitlab,omitempty"`
}

type GitHubSettings struct {
	Token string `yaml:"token,omitempty"`
}

// GitLabSettings holds per-host tokens. key is the gitlab hostname
// (e.g. "gitlab.com", "gitlab.archlinux.org") so self-hosted instances
// can each have their own credential.
type GitLabSettings struct {
	Tokens map[string]string `yaml:"tokens,omitempty"`
}

func defaultConfig() *KonfConfig {
	return &KonfConfig{
		Theme:    "catppuccin",
		LogLevel: "info",
		NerdFont: true,
	}
}

// InitConfig loads ~/.config/konfi/config.yaml or creates defaults.
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
