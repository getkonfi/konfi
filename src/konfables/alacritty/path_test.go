package alacritty

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPathUsesExistingAlternateXDGFile(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(home, "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	want := filepath.Join(xdg, "alacritty.toml")
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := DefaultConfigPath()
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDefaultConfigPathDefaultsToPrimaryXDGFile(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(home, "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	got := DefaultConfigPath()
	want := filepath.Join(xdg, "alacritty", "alacritty.toml")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
