package alacritty

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
		{"font.size", "12.0", true},
		{"font.normal.family", "JetBrains Mono", true},
		{"colors.primary.background", "#282828", true},
		{"colors.primary.foreground", "#ebdbb2", true},
		{"window.opacity", "1.0", true},
		{"window.padding.x", "8", true},
		{"window.padding.y", "8", true},
		{"nonexistent", "", false},
		{"font.missing", "", false},
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
		{"font.size", 3, true},
		{"font.normal.family", 6, true},
		{"colors.primary.background", 9, true},
		{"window.opacity", 13, true},
		{"window.padding.x", 16, true},
		{"nonexistent", -1, false},
		{"font.missing", -1, false},
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
		{"replace nested value", "font.normal.family", "\"Fira Code\"", "set_nested.txt"},
		{"add to shallow section", "window.title", "\"Terminal\"", "set_shallow.txt"},
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
		{"delete nested key", "colors.primary.background", "delete_nested.txt"},
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
