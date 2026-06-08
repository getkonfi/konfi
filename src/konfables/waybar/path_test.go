package waybar

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPathUsesXDGConfigHome(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(home, "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	got := DefaultConfigPath()
	want := filepath.Join(xdg, "waybar", "config")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDefaultConfigPathFallsBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	got := DefaultConfigPath()
	want := filepath.Join(home, ".config", "waybar", "config")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDefaultConfigPathUsesExistingJSONCConfig(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(home, "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	want := filepath.Join(xdg, "waybar", "config.jsonc")
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := DefaultConfigPath(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDefaultConfigPathPrefersExistingConfig(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(home, "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	config := filepath.Join(xdg, "waybar", "config")
	jsonc := filepath.Join(xdg, "waybar", "config.jsonc")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{config, jsonc} {
		if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if got := DefaultConfigPath(); got != config {
		t.Fatalf("got %q, want %q", got, config)
	}
}
