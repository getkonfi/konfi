package gnome

import (
	"testing"
)

func TestParseFlat(t *testing.T) {
	data := []byte(`org.gnome.desktop.interface/color-scheme = prefer-dark
org.gnome.desktop.interface/gtk-theme = Adwaita
# comment line
org.gnome.desktop.background/primary-color = #023c88
`)

	m := parseFlat(data)
	if len(m) != 3 {
		t.Fatalf("got %d entries, want 3", len(m))
	}
	if m["org.gnome.desktop.interface/color-scheme"] != "prefer-dark" {
		t.Errorf("color-scheme = %q", m["org.gnome.desktop.interface/color-scheme"])
	}
	if m["org.gnome.desktop.background/primary-color"] != "#023c88" {
		t.Errorf("primary-color = %q", m["org.gnome.desktop.background/primary-color"])
	}
}

func TestParseFlatEmpty(t *testing.T) {
	m := parseFlat([]byte(""))
	if len(m) != 0 {
		t.Errorf("expected empty map, got %d entries", len(m))
	}
}

func TestParseFlatSkipsComments(t *testing.T) {
	data := []byte("# this is a comment\nkey = val\n")
	m := parseFlat(data)
	if len(m) != 1 {
		t.Errorf("expected 1 entry, got %d", len(m))
	}
}

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

func TestDiffOnlySetsChangedKeys(t *testing.T) {
	original := []byte(`org.gnome.desktop.interface/color-scheme = default
org.gnome.desktop.interface/cursor-size = 24
org.gnome.desktop.interface/gtk-theme = Adwaita
`)
	data := []byte(`org.gnome.desktop.interface/color-scheme = prefer-dark
org.gnome.desktop.interface/cursor-size = 24
org.gnome.desktop.interface/gtk-theme = Adwaita
`)

	origMap := parseFlat(original)
	newMap := parseFlat(data)

	// only color-scheme should be different
	var changedKeys []string
	for key, newVal := range newMap {
		if origVal, ok := origMap[key]; ok && origVal == newVal {
			continue
		}
		changedKeys = append(changedKeys, key)
	}

	if len(changedKeys) != 1 {
		t.Fatalf("expected 1 changed key, got %d: %v", len(changedKeys), changedKeys)
	}
	if changedKeys[0] != "org.gnome.desktop.interface/color-scheme" {
		t.Errorf("expected color-scheme, got %q", changedKeys[0])
	}
}

func TestCutKV(t *testing.T) {
	tests := []struct {
		input string
		key   string
		val   string
		ok    bool
	}{
		{"org.gnome.desktop.interface/color-scheme = prefer-dark", "org.gnome.desktop.interface/color-scheme", "prefer-dark", true},
		{"key = value with spaces", "key", "value with spaces", true},
		{"no-equals-sign", "", "", false},
		{"key=no-spaces", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			k, v, ok := cutKV(tt.input)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if k != tt.key {
				t.Errorf("key = %q, want %q", k, tt.key)
			}
			if v != tt.val {
				t.Errorf("val = %q, want %q", v, tt.val)
			}
		})
	}
}
