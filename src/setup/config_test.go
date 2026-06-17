package setup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/getkonfi/konfi/setup/cst"
)

func TestDefaultConfigDisablesNerdFont(t *testing.T) {
	cfg := defaultConfig()
	if cfg.NerdFont {
		t.Fatal("default config enables nerd font icons")
	}
}

func TestInitConfigMissingAndUnspecifiedNerdFontFallback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))

	app := &App{}
	if err := InitConfig(context.Background(), app); err != nil {
		t.Fatal(err)
	}
	if app.Config.NerdFont {
		t.Fatal("missing config enables nerd font icons")
	}

	if err := os.MkdirAll(cst.ConfigDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cst.ConfigFilePath(), []byte("theme: catppuccin\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	app = &App{}
	if err := InitConfig(context.Background(), app); err != nil {
		t.Fatal(err)
	}
	if app.Config.NerdFont {
		t.Fatal("config without nerd_font enables nerd font icons")
	}
}

func TestInitConfigPreservesExplicitNerdFont(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))

	if err := os.MkdirAll(cst.ConfigDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cst.ConfigFilePath(), []byte("theme: catppuccin\nnerd_font: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := &App{}
	if err := InitConfig(context.Background(), app); err != nil {
		t.Fatal(err)
	}
	if !app.Config.NerdFont {
		t.Fatal("explicit nerd_font true was not preserved")
	}
}
