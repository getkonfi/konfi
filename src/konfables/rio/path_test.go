package rio

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfigPathUsesRioConfigHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIO_CONFIG_HOME", dir)

	got := DefaultConfigPath()
	want := filepath.Join(dir, "config.toml")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
