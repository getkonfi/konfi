package gnome

import (
	"testing"
)

func TestSplitFlatKey(t *testing.T) {
	tests := []struct {
		input      string
		wantSchema string
		wantKey    string
		wantOK     bool
	}{
		{"org.gnome.desktop.interface/color-scheme", "org.gnome.desktop.interface", "color-scheme", true},
		{"org.gnome.desktop.background/picture-uri", "org.gnome.desktop.background", "picture-uri", true},
		{"no-slash-key", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			schema, key, ok := splitFlatKey(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if schema != tt.wantSchema {
				t.Errorf("schema = %q, want %q", schema, tt.wantSchema)
			}
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q", key, tt.wantKey)
			}
		})
	}
}

func TestStripQuotes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"'prefer-dark'", "prefer-dark"},
		{"'Adwaita'", "Adwaita"},
		{"24", "24"},
		{"'value with spaces'", "value with spaces"},
		{"''", ""},
		{"'a'", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripQuotes(tt.input)
			if got != tt.want {
				t.Errorf("stripQuotes(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
