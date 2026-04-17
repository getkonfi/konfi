package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/emin/konfigurator/setup"
	"github.com/emin/konfigurator/setup/cst"

	"gopkg.in/yaml.v3"
)

// configSearchPaths returns the ordered list of files to merge into config.
// later paths override earlier ones at the top-level field granularity
// (nested maps like gitlab.tokens are replaced wholesale, not deep-merged —
// keep your tokens in one file).
//
// order:
//  1. ~/.config/konfi/config.yaml  — deployed user config
//  2. ./config.yaml                — repo-committed dev base
//  3. ./config.local.yaml          — gitignored dev overrides
func configSearchPaths() []string {
	return []string{
		cst.ConfigFilePath(),
		"config.yaml",
		"config.local.yaml",
	}
}

// loadConfig reads every path that exists and overlays them in order.
func loadConfig() (*setup.KonfConfig, []string, error) {
	cfg := &setup.KonfConfig{}
	var loaded []string

	for _, p := range configSearchPaths() {
		data, err := os.ReadFile(p)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, loaded, fmt.Errorf("read %s: %w", p, err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, loaded, fmt.Errorf("parse %s: %w", p, err)
		}
		loaded = append(loaded, p)
	}

	return cfg, loaded, nil
}

// gitlabTokenFor returns the token for a gitlab host, or "" if not configured.
func gitlabTokenFor(cfg *setup.KonfConfig, host string) string {
	if cfg == nil || cfg.Upstream == nil || cfg.Upstream.GitLab == nil {
		return ""
	}
	return cfg.Upstream.GitLab.Tokens[host]
}

// githubToken returns the configured github token, or "".
func githubToken(cfg *setup.KonfConfig) string {
	if cfg == nil || cfg.Upstream == nil || cfg.Upstream.GitHub == nil {
		return ""
	}
	return cfg.Upstream.GitHub.Token
}
