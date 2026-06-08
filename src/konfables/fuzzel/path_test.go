package fuzzel

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfigPathUsesXDGConfigHome(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(home, "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	got := DefaultConfigPath()
	want := filepath.Join(xdg, "fuzzel", "fuzzel.ini")
	if got != want {
		t.Fatalf("DefaultConfigPath() = %q, want %q", got, want)
	}
}

func TestDefaultConfigPathFallsBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	got := DefaultConfigPath()
	want := filepath.Join(home, ".config", "fuzzel", "fuzzel.ini")
	if got != want {
		t.Fatalf("DefaultConfigPath() = %q, want %q", got, want)
	}
}
