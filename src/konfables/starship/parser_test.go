package starship

import (
	"bytes"
	"os"
	"testing"
)

func loadTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read testdata/%s: %v", name, err)
	}
	return data
}

func TestFindValue(t *testing.T) {
	p := &parser{}
	data := loadTestdata(t, "config.txt")

	tests := []struct {
		key    string
		want   string
		wantOK bool
	}{
		{"format", "$all", true},
		{"scan_timeout", "30", true},
		{"character.success_symbol", "[➜](bold green)", true},
		{"git_branch.style", "bold purple", true},
		{"package.disabled", "true", true},
		{"nonexistent", "", false},
		{"character.missing", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := p.FindValue(data, tt.key)
			if ok != tt.wantOK {
				t.Fatalf("FindValue(%q) ok = %v, want %v", tt.key, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("FindValue(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestFindLine(t *testing.T) {
	p := &parser{}
	data := loadTestdata(t, "config.txt")

	tests := []struct {
		key    string
		want   int
		wantOK bool
	}{
		{"format", 1, true},
		{"scan_timeout", 2, true},
		{"character.success_symbol", 5, true},
		{"git_branch.style", 9, true},
		{"package.disabled", 12, true},
		{"nonexistent", -1, false},
		{"character.missing", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := p.FindLine(data, tt.key)
			if ok != tt.wantOK {
				t.Fatalf("FindLine(%q) ok = %v, want %v", tt.key, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("FindLine(%q) = %d, want %d", tt.key, got, tt.want)
			}
		})
	}
}

func TestSetValue(t *testing.T) {
	p := &parser{}

	tests := []struct {
		name   string
		key    string
		value  string
		golden string
	}{
		{"replace section key", "character.success_symbol", "\"[→](bold cyan)\"", "set_section.txt"},
		{"add top-level key", "add_newline", "false", "set_toplevel.txt"},
		{"add to existing section", "git_branch.truncation_length", "10", "set_add_section.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := loadTestdata(t, "config.txt")
			got, err := p.SetValue(data, tt.key, tt.value)
			if err != nil {
				t.Fatalf("SetValue(%q, %q): %v", tt.key, tt.value, err)
			}
			want := loadTestdata(t, tt.golden)
			if !bytes.Equal(got, want) {
				t.Errorf("SetValue(%q) mismatch\ngot:\n%s\nwant:\n%s", tt.key, got, want)
			}
		})
	}
}

func TestDeleteKey(t *testing.T) {
	p := &parser{}

	tests := []struct {
		name   string
		key    string
		golden string
	}{
		{"delete section key", "character.error_symbol", "delete_section.txt"},
		{"delete top-level key", "scan_timeout", "delete_toplevel.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := loadTestdata(t, "config.txt")
			got, err := p.DeleteKey(data, tt.key)
			if err != nil {
				t.Fatalf("DeleteKey(%q): %v", tt.key, err)
			}
			want := loadTestdata(t, tt.golden)
			if !bytes.Equal(got, want) {
				t.Errorf("DeleteKey(%q) mismatch\ngot:\n%s\nwant:\n%s", tt.key, got, want)
			}
		})
	}
}

func TestDeleteKeyMissing(t *testing.T) {
	p := &parser{}
	data := loadTestdata(t, "config.txt")

	got, err := p.DeleteKey(data, "nonexistent.key")
	if err != nil {
		t.Fatalf("DeleteKey(missing): %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Error("DeleteKey(missing) should return data unchanged")
	}
}
