package kitty

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfigPathUsesKittyConfigDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KITTY_CONFIG_DIRECTORY", dir)

	got := DefaultConfigPath()
	want := filepath.Join(dir, "kitty.conf")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
