package powerlevel10k

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	t.Setenv("POWERLEVEL9K_CONFIG_FILE", "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	want := filepath.Join(home, ".p10k.zsh")
	if got := DefaultConfigPath(); got != want {
		t.Fatalf("DefaultConfigPath() = %q, want %q", got, want)
	}
}

func TestDefaultConfigPathEnvOverride(t *testing.T) {
	t.Setenv("POWERLEVEL9K_CONFIG_FILE", "/tmp/custom-p10k.zsh")
	if got := DefaultConfigPath(); got != "/tmp/custom-p10k.zsh" {
		t.Fatalf("DefaultConfigPath env = %q", got)
	}
}
