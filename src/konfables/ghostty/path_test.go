package ghostty

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPathDarwinDefaultsNative(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	got := defaultConfigPath("darwin")
	want := filepath.Join(home, "Library", "Application Support", "com.mitchellh.ghostty", "config.ghostty")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDefaultConfigPathDarwinUsesHighestPrecedenceExisting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	xdg := filepath.Join(home, ".config", "ghostty", "config.ghostty")
	native := filepath.Join(home, "Library", "Application Support", "com.mitchellh.ghostty", "config")
	for _, path := range []string{xdg, native} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := defaultConfigPath("darwin")
	if got != native {
		t.Fatalf("got %q, want %q", got, native)
	}
}

func TestDefaultConfigPathUsesExplicitXDGOnDarwin(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(home, "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	got := defaultConfigPath("darwin")
	want := filepath.Join(xdg, "ghostty", "config.ghostty")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
