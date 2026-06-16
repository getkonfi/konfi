package gnome

import (
	"sort"
	"testing"

	"github.com/getkonfi/konfi/pkg"
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

func TestManagedKeysMatchSchema(t *testing.T) {
	s, err := pkg.LoadSchema(schemaData)
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}

	schemaKeys := s.SchemaKeys()
	managed := make(map[string]struct{}, len(managedKeys))
	for _, mk := range managedKeys {
		managed[mk.Schema+"/"+mk.Key] = struct{}{}
	}

	var missing []string
	for key := range schemaKeys {
		if _, ok := managed[key]; !ok {
			missing = append(missing, key)
		}
	}
	var extra []string
	for key := range managed {
		if _, ok := schemaKeys[key]; !ok {
			extra = append(extra, key)
		}
	}

	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 || len(extra) > 0 {
		t.Fatalf("managed keys drifted from schema: missing=%v extra=%v", missing, extra)
	}
}

func TestScalingFactorGVariantNormalization(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"uint32 0", "0"},
		{"uint32 2", "2"},
		{"uint64 3", "3"},
		{"0", "0"},
	}

	for _, tt := range tests {
		got := normalizeGSettingsValue("org.gnome.desktop.interface", "scaling-factor", tt.input)
		if got != tt.want {
			t.Errorf("normalizeGSettingsValue(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}

	got := normalizeGSettingsValue("org.gnome.desktop.interface", "cursor-size", "uint32 24")
	if got != "uint32 24" {
		t.Errorf("non-scaling key normalized to %q", got)
	}
}

func TestScalingFactorGVariantSerialization(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0", "uint32 0"},
		{"2", "uint32 2"},
		{"uint32 3", "uint32 3"},
	}

	for _, tt := range tests {
		got := serializeGSettingsValue("org.gnome.desktop.interface", "scaling-factor", tt.input)
		if got != tt.want {
			t.Errorf("serializeGSettingsValue(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}

	got := serializeGSettingsValue("org.gnome.desktop.interface", "cursor-size", "24")
	if got != "24" {
		t.Errorf("non-scaling key serialized to %q", got)
	}
}
