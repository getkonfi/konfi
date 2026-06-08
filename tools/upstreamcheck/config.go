package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/eminert/konfi/setup/cst"

	"gopkg.in/yaml.v3"
)

type upstreamConfig struct {
	Upstream *upstreamSettings `yaml:"upstream,omitempty"`
}

type upstreamSettings struct {
	GitHub *githubSettings `yaml:"github,omitempty"`
	GitLab *gitlabSettings `yaml:"gitlab,omitempty"`
}

type githubSettings struct {
	Token string `yaml:"token,omitempty"`
}

type gitlabSettings struct {
	Tokens map[string]string `yaml:"tokens,omitempty"`
}

// configSearchPaths returns the ordered list of files to merge into config.
// later paths override earlier ones at the top-level field granularity
// (nested maps like gitlab.tokens are replaced wholesale, not deep-merged —
// keep your tokens in one file).
//
// order:
//  1. platform config dir           - deployed user config
//  2. ./config.yaml, ../config.yaml, or ../../config.yaml - repo dev base
//  3. ./config.local.yaml, ../config.local.yaml, or ../../config.local.yaml - local dev overrides
func configSearchPaths() []string {
	return []string{
		cst.ConfigFilePath(),
		"config.yaml",
		filepath.Join("..", "config.yaml"),
		filepath.Join("..", "..", "config.yaml"),
		"config.local.yaml",
		filepath.Join("..", "config.local.yaml"),
		filepath.Join("..", "..", "config.local.yaml"),
	}
}

// loadConfig reads every path that exists and overlays them in order.
func loadConfig() (*upstreamConfig, []string, error) {
	cfg := &upstreamConfig{}
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
func gitlabTokenFor(cfg *upstreamConfig, host string) string {
	if cfg == nil || cfg.Upstream == nil || cfg.Upstream.GitLab == nil {
		return ""
	}
	return cfg.Upstream.GitLab.Tokens[host]
}

// githubToken returns the configured github token, or "".
func githubToken(cfg *upstreamConfig) string {
	if cfg == nil || cfg.Upstream == nil || cfg.Upstream.GitHub == nil {
		return ""
	}
	return cfg.Upstream.GitHub.Token
}
