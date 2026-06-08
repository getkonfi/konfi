package tmux

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPathPrefersDotTmuxConf(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(home, "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	dotfile := filepath.Join(home, ".tmux.conf")
	xdgConfig := filepath.Join(xdg, "tmux", "tmux.conf")
	for _, path := range []string{dotfile, xdgConfig} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := DefaultConfigPath()
	if got != dotfile {
		t.Fatalf("got %q, want %q", got, dotfile)
	}
}

func TestDefaultConfigPathUsesExistingXDGConfig(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(home, "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	want := filepath.Join(xdg, "tmux", "tmux.conf")
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
