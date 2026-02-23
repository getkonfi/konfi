package pkg

import (
	"os"
	"testing"
)

func loadTestTOML(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/toml/basic.toml")
	if err != nil {
		t.Fatalf("read test toml: %v", err)
	}
	return data
}

func TestFindTopLevelKey(t *testing.T) {
	data := loadTestTOML(t)

	tests := []struct {
		key   string
		want  string
		found bool
	}{
		{"title", "TOML Example", true},
		{"debug", "false", true},
		{"missing", "", false},
	}

	for _, tt := range tests {
		got, _, found := FindTopLevelKey(data, tt.key)
		if found != tt.found {
			t.Errorf("FindTopLevelKey(%q): found=%v, want=%v", tt.key, found, tt.found)
		}
		if got != tt.want {
			t.Errorf("FindTopLevelKey(%q): got=%q, want=%q", tt.key, got, tt.want)
		}
	}
}

func TestFindKeyInSection(t *testing.T) {
	data := loadTestTOML(t)

	tests := []struct {
		section string
		key     string
		want    string
		found   bool
	}{
		{"owner", "name", "Tom Preston-Werner", true},
		{"database", "server", "192.168.1.1", true},
		{"database", "enabled", "true", true},
		{"servers.alpha", "ip", "10.0.0.1", true},
		{"database", "missing", "", false},
		{"nosection", "name", "", false},
	}

	for _, tt := range tests {
		got, _, found := FindKeyInSection(data, tt.section, tt.key)
		if found != tt.found {
			t.Errorf("FindKeyInSection(%q, %q): found=%v, want=%v", tt.section, tt.key, found, tt.found)
		}
		if got != tt.want {
			t.Errorf("FindKeyInSection(%q, %q): got=%q, want=%q", tt.section, tt.key, got, tt.want)
		}
	}
}

func TestReplaceValueOnLine(t *testing.T) {
	data := loadTestTOML(t)

	// find the server line and replace it
	_, lineIdx, found := FindKeyInSection(data, "database", "server")
	if !found {
		t.Fatal("expected to find database.server")
	}

	result := ReplaceValueOnLine(data, lineIdx, `"10.0.0.1"`)
	got, _, found := FindKeyInSection(result, "database", "server")
	if !found {
		t.Fatal("expected to find replaced value")
	}
	if got != "10.0.0.1" {
		t.Errorf("got %q, want %q", got, "10.0.0.1")
	}
}

func TestDeleteKeyOnLine(t *testing.T) {
	data := loadTestTOML(t)

	_, lineIdx, found := FindKeyInSection(data, "database", "enabled")
	if !found {
		t.Fatal("expected to find database.enabled")
	}

	result := DeleteKeyOnLine(data, lineIdx)
	_, _, found = FindKeyInSection(result, "database", "enabled")
	if found {
		t.Error("expected database.enabled to be deleted")
	}

	// other keys should survive
	_, _, found = FindKeyInSection(result, "database", "server")
	if !found {
		t.Error("expected database.server to survive deletion")
	}
}

func TestInsertKeyInSection(t *testing.T) {
	data := loadTestTOML(t)

	result := InsertKeyInSection(data, "database", "timeout", "30")
	got, _, found := FindKeyInSection(result, "database", "timeout")
	if !found {
		t.Fatal("expected to find inserted key")
	}
	if got != "30" {
		t.Errorf("got %q, want %q", got, "30")
	}

	// existing keys should survive
	_, _, found = FindKeyInSection(result, "database", "server")
	if !found {
		t.Error("expected existing keys to survive")
	}
}

func TestInsertKeyInNewSection(t *testing.T) {
	data := loadTestTOML(t)

	result := InsertKeyInSection(data, "logging", "level", "info")
	got, _, found := FindKeyInSection(result, "logging", "level")
	if !found {
		t.Fatal("expected to find key in new section")
	}
	if got != "info" {
		t.Errorf("got %q, want %q", got, "info")
	}
}

func TestRoundTrip(t *testing.T) {
	data := loadTestTOML(t)

	// replace a value then read it back
	_, lineIdx, _ := FindKeyInSection(data, "owner", "name")
	result := ReplaceValueOnLine(data, lineIdx, `"New Name"`)
	got, _, found := FindKeyInSection(result, "owner", "name")
	if !found || got != "New Name" {
		t.Errorf("round-trip failed: found=%v got=%q", found, got)
	}

	// untouched keys should be identical
	origTitle, _, _ := FindTopLevelKey(data, "title")
	newTitle, _, _ := FindTopLevelKey(result, "title")
	if origTitle != newTitle {
		t.Error("untouched key was modified")
	}
}
