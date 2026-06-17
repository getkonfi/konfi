package selfupdate

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArtifactName(t *testing.T) {
	got, err := artifactName("konfi", "v1.2.3", "linux", "amd64")
	if err != nil {
		t.Fatalf("artifactName returned error: %v", err)
	}
	if got != "konfi_1.2.3_linux_amd64.tar.gz" {
		t.Fatalf("artifactName = %q", got)
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		current string
		target  string
		want    int
	}{
		{"1.2.3", "1.2.3", 0},
		{"1.2.3-dev", "1.2.3", -1},
		{"1.2.4", "1.2.3", 1},
		{"v1.2.2", "1.2.3", -1},
	}

	for _, tt := range tests {
		got, ok := compareVersions(tt.current, tt.target)
		if !ok {
			t.Fatalf("compareVersions(%q, %q) not ok", tt.current, tt.target)
		}
		if sign(got) != tt.want {
			t.Fatalf("compareVersions(%q, %q) = %d, want sign %d", tt.current, tt.target, got, tt.want)
		}
	}
}

func TestVerifyArchiveChecksum(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "konfi_1.2.3_linux_amd64.tar.gz")
	if err := os.WriteFile(archive, []byte("archive"), 0o644); err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256([]byte("archive"))
	checksums := filepath.Join(dir, "checksums.txt")
	line := fmt.Sprintf("%x  %s\n", sum, filepath.Base(archive))
	if err := os.WriteFile(checksums, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifyArchiveChecksum(archive, checksums, filepath.Base(archive)); err != nil {
		t.Fatalf("verifyArchiveChecksum returned error: %v", err)
	}
}

func TestDetectManagedInstallByPath(t *testing.T) {
	tests := []struct {
		path    string
		manager string
	}{
		{"/nix/store/abc-konfi-1.2.3/bin/konfi", "nix"},
		{"/opt/homebrew/Caskroom/konfi/1.2.3/konfi", "homebrew"},
		{"/usr/local/Cellar/konfi/1.2.3/bin/konfi", "homebrew"},
	}

	for _, tt := range tests {
		got, ok := managedInstallByPath(tt.path)
		if !ok {
			t.Fatalf("managedInstallByPath(%q) not ok", tt.path)
		}
		if got.Manager != tt.manager {
			t.Fatalf("managedInstallByPath(%q) manager = %q", tt.path, got.Manager)
		}
	}
}

func TestManagedInstallErrorMentionsCommand(t *testing.T) {
	err := (&ManagedInstall{Manager: "homebrew", Command: "brew upgrade konfi", Path: "/opt/homebrew/bin/konfi"}).Error()
	if !strings.Contains(err, "brew upgrade konfi") {
		t.Fatalf("error did not include update command: %q", err)
	}
}

func sign(v int) int {
	switch {
	case v < 0:
		return -1
	case v > 0:
		return 1
	default:
		return 0
	}
}
