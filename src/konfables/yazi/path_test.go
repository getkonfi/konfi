package yazi

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfigPathUsesXDGConfigHome(t *testing.T) {
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("YAZI_CONFIG_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", xdg)

	got := DefaultConfigPath()
	want := filepath.Join(xdg, "yazi", "yazi.toml")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDefaultConfigPathFallsBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("YAZI_CONFIG_HOME", "")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	got := DefaultConfigPath()
	want := filepath.Join(home, ".config", "yazi", "yazi.toml")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDefaultConfigPathUsesYaziConfigHome(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "yazi-alt")
	t.Setenv("YAZI_CONFIG_HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))

	got := DefaultConfigPath()
	want := filepath.Join(dir, "yazi.toml")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
