package brew

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPathUsesGlobalEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Brewfile")
	t.Setenv("HOMEBREW_BUNDLE_FILE_GLOBAL", path)

	got := DefaultConfigPath()
	if got != path {
		t.Fatalf("got %q, want %q", got, path)
	}
}

func TestDefaultConfigPathUsesXDGHomebrewBrewfile(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(home, "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOMEBREW_BUNDLE_FILE_GLOBAL", "")

	got := DefaultConfigPath()
	want := filepath.Join(xdg, "homebrew", "Brewfile")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDefaultConfigPathUsesExistingHomebrewDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOMEBREW_BUNDLE_FILE_GLOBAL", "")

	want := filepath.Join(home, ".homebrew", "Brewfile")
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
