package starship

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfigPathUsesStarshipConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prompt.toml")
	t.Setenv("STARSHIP_CONFIG", path)

	got := DefaultConfigPath()
	if got != path {
		t.Fatalf("got %q, want %q", got, path)
	}
}
