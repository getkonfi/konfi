package pkg

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadSchema(t *testing.T) {
	// test against all embedded schema files
	schemas := []struct {
		path string
		app  string
	}{
		{"../konfables/ghostty/schema.yaml", "ghostty"},
		{"../konfables/starship/schema.yaml", "starship"},
		{"../konfables/alacritty/schema.yaml", "alacritty"},
		{"../konfables/hyprland/schema.yaml", "hyprland"},
	}

	for _, tt := range schemas {
		t.Run(tt.app, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Clean(tt.path))
			if err != nil {
				t.Fatalf("read schema: %v", err)
			}

			s, err := LoadSchema(data)
			if err != nil {
				t.Fatalf("parse schema: %v", err)
			}

			if s.App != tt.app {
				t.Errorf("app: got %q, want %q", s.App, tt.app)
			}

			if len(s.Sections) == 0 {
				t.Error("expected at least one section")
			}

			// ghostty should have 50+ fields after schema expansion
			if tt.app == "ghostty" {
				total := 0
				for _, sec := range s.Sections {
					total += len(sec.Fields)
				}
				if total < 50 {
					t.Errorf("ghostty schema too small: got %d fields, want at least 50", total)
				}
				if len(s.Sections) < 14 {
					t.Errorf("ghostty schema too few sections: got %d, want at least 14", len(s.Sections))
				}
			}

			// enriched app schemas should meet minimum field counts
			if tt.app == "alacritty" {
				total := 0
				for _, sec := range s.Sections {
					total += len(sec.Fields)
				}
				if total < 40 {
					t.Errorf("alacritty schema too small: got %d fields, want at least 40", total)
				}
			}
			if tt.app == "hyprland" {
				total := 0
				for _, sec := range s.Sections {
					total += len(sec.Fields)
				}
				if total < 50 {
					t.Errorf("hyprland schema too small: got %d fields, want at least 50", total)
				}
			}
			if tt.app == "starship" {
				total := 0
				for _, sec := range s.Sections {
					total += len(sec.Fields)
				}
				if total < 60 {
					t.Errorf("starship schema too small: got %d fields, want at least 60", total)
				}
			}

			// verify every field has required attributes
			for _, sec := range s.Sections {
				if sec.Name == "" {
					t.Error("section missing name")
				}
				for _, f := range sec.Fields {
					if f.Key == "" {
						t.Errorf("section %q: field missing key", sec.Name)
					}
					if f.Label == "" {
						t.Errorf("section %q, field %q: missing label", sec.Name, f.Key)
					}
					if f.Type == "" {
						t.Errorf("section %q, field %q: missing type", sec.Name, f.Key)
					}
				}
			}
		})
	}
}

func TestLoadSchema_EnrichedFields(t *testing.T) {
	raw := `
app: test
format: toml
docs_url: "https://example.com/docs"
sections:
  - name: General
    key: ""
    fields:
      - key: font
        label: Font
        type: string
        default: mono
        description: primary font
        example: 'font = "JetBrains Mono"'
        hint: use a monospace font
        doc_url: "https://example.com/docs/font"
        since: "1.0.0"
      - key: legacy
        label: Legacy
        type: bool
        default: "false"
        description: deprecated option
        since: "0.5.0"
        until: "2.0.0"
`
	s, err := LoadSchema([]byte(raw))
	if err != nil {
		t.Fatalf("parse enriched schema: %v", err)
	}

	if s.DocsURL != "https://example.com/docs" {
		t.Errorf("docs_url: got %q, want %q", s.DocsURL, "https://example.com/docs")
	}

	f := s.Sections[0].Fields[0]
	if f.Example != `font = "JetBrains Mono"` {
		t.Errorf("example: got %q", f.Example)
	}
	if f.Hint != "use a monospace font" {
		t.Errorf("hint: got %q", f.Hint)
	}
	if f.DocURL != "https://example.com/docs/font" {
		t.Errorf("doc_url: got %q", f.DocURL)
	}
	if f.Since != "1.0.0" {
		t.Errorf("since: got %q", f.Since)
	}
	if f.Until != "" {
		t.Errorf("until: expected empty, got %q", f.Until)
	}

	legacy := s.Sections[0].Fields[1]
	if legacy.Since != "0.5.0" {
		t.Errorf("legacy since: got %q", legacy.Since)
	}
	if legacy.Until != "2.0.0" {
		t.Errorf("legacy until: got %q", legacy.Until)
	}
}

func TestLoadSchema_EnrichedFieldsOmitEmpty(t *testing.T) {
	// fields without enriched metadata should marshal without those keys
	f := Field{
		Key:     "test",
		Label:   "Test",
		Type:    "string",
		Default: "",
	}
	data, err := yaml.Marshal(f)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	for _, key := range []string{"example:", "hint:", "doc_url:", "since:", "until:"} {
		if contains(s, key) {
			t.Errorf("empty field should omit %q, got:\n%s", key, s)
		}
	}
}

// --- FilterByVersion tests ---

func makeVersionSchema() *Schema {
	return &Schema{
		App:     "test",
		Format:  "toml",
		DocsURL: "https://example.com/docs",
		Sections: []Section{
			{
				Name: "General",
				Key:  "",
				Fields: []Field{
					{Key: "always", Label: "Always", Type: "string"},
					{Key: "new-field", Label: "New", Type: "string", Since: "2.0.0"},
					{Key: "deprecated", Label: "Old", Type: "bool", Until: "2.0.0"},
					{Key: "window", Label: "Window", Type: "string", Since: "1.0.0", Until: "3.0.0"},
				},
			},
			{
				Name: "Legacy",
				Key:  "legacy",
				Fields: []Field{
					{Key: "removed", Label: "Removed", Type: "string", Until: "1.0.0"},
				},
			},
		},
	}
}

func TestFilterByVersion_EmptyVersion(t *testing.T) {
	s := makeVersionSchema()
	got := s.FilterByVersion("")
	if len(got.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(got.Sections))
	}
	total := 0
	for _, sec := range got.Sections {
		total += len(sec.Fields)
	}
	if total != 5 {
		t.Errorf("expected 5 fields, got %d", total)
	}
	if got.DocsURL != s.DocsURL {
		t.Errorf("docs_url not copied")
	}
}

func TestFilterByVersion_SinceHides(t *testing.T) {
	s := makeVersionSchema()
	got := s.FilterByVersion("v1.0.0")
	for _, sec := range got.Sections {
		for _, f := range sec.Fields {
			if f.Key == "new-field" {
				t.Errorf("new-field (since=2.0.0) should be hidden at v1.0.0")
			}
		}
	}
}

func TestFilterByVersion_SinceShows(t *testing.T) {
	s := makeVersionSchema()
	got := s.FilterByVersion("v2.0.0")
	found := false
	for _, sec := range got.Sections {
		for _, f := range sec.Fields {
			if f.Key == "new-field" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("new-field (since=2.0.0) should be visible at v2.0.0")
	}
}

func TestFilterByVersion_UntilHides(t *testing.T) {
	s := makeVersionSchema()
	got := s.FilterByVersion("v2.0.0")
	for _, sec := range got.Sections {
		for _, f := range sec.Fields {
			if f.Key == "deprecated" {
				t.Errorf("deprecated (until=2.0.0) should be hidden at v2.0.0 (exclusive)")
			}
		}
	}
}

func TestFilterByVersion_UntilShows(t *testing.T) {
	s := makeVersionSchema()
	got := s.FilterByVersion("v1.9.0")
	found := false
	for _, sec := range got.Sections {
		for _, f := range sec.Fields {
			if f.Key == "deprecated" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("deprecated (until=2.0.0) should be visible at v1.9.0")
	}
}

func TestFilterByVersion_CombinedSinceUntil(t *testing.T) {
	s := makeVersionSchema()

	// visible inside the window
	got := s.FilterByVersion("v2.0.0")
	found := false
	for _, sec := range got.Sections {
		for _, f := range sec.Fields {
			if f.Key == "window" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("window (1.0.0-3.0.0) should be visible at v2.0.0")
	}

	// hidden before since
	got = s.FilterByVersion("v0.5.0")
	for _, sec := range got.Sections {
		for _, f := range sec.Fields {
			if f.Key == "window" {
				t.Errorf("window (since=1.0.0) should be hidden at v0.5.0")
			}
		}
	}

	// hidden at until boundary
	got = s.FilterByVersion("v3.0.0")
	for _, sec := range got.Sections {
		for _, f := range sec.Fields {
			if f.Key == "window" {
				t.Errorf("window (until=3.0.0) should be hidden at v3.0.0")
			}
		}
	}
}

func TestFilterByVersion_InvalidSemverPassthrough(t *testing.T) {
	s := &Schema{
		App:    "test",
		Format: "toml",
		Sections: []Section{
			{
				Name: "General",
				Fields: []Field{
					{Key: "bad-since", Label: "Bad", Type: "string", Since: "not-semver"},
					{Key: "bad-until", Label: "Bad2", Type: "string", Until: "also-bad"},
				},
			},
		},
	}
	got := s.FilterByVersion("v1.0.0")
	if len(got.Sections) != 1 || len(got.Sections[0].Fields) != 2 {
		t.Errorf("invalid semver in since/until should be treated as unset — expected 2 fields, got %d",
			len(got.Sections[0].Fields))
	}
}

func TestFilterByVersion_EmptySectionDropped(t *testing.T) {
	s := makeVersionSchema()
	// at v1.0.0+, "Legacy" section has only "removed" (until=1.0.0) which gets filtered out
	got := s.FilterByVersion("v1.5.0")
	for _, sec := range got.Sections {
		if sec.Name == "Legacy" {
			t.Errorf("Legacy section should be dropped (all fields filtered out)")
		}
	}
}

func TestFilterByVersion_NoMutation(t *testing.T) {
	s := makeVersionSchema()
	origLen := len(s.Sections)
	_ = s.FilterByVersion("v5.0.0")
	if len(s.Sections) != origLen {
		t.Errorf("original schema was mutated")
	}
}

func TestFilterByVersion_WithoutVPrefix(t *testing.T) {
	s := makeVersionSchema()
	got := s.FilterByVersion("1.0.0")
	for _, sec := range got.Sections {
		for _, f := range sec.Fields {
			if f.Key == "new-field" {
				t.Errorf("new-field (since=2.0.0) should be hidden at 1.0.0 (no v prefix)")
			}
		}
	}
}

func TestSchemaKeys(t *testing.T) {
	s := makeVersionSchema()
	keys := s.SchemaKeys()

	if len(keys) != 5 {
		t.Errorf("expected 5 schema keys, got %d", len(keys))
	}

	// no dupes possible in a map, but verify expected keys exist
	expected := []string{"always", "new-field", "deprecated", "window", "removed"}
	for _, k := range expected {
		if _, ok := keys[k]; !ok {
			t.Errorf("missing expected key %q", k)
		}
	}
}

func TestDiagnose_Unknown(t *testing.T) {
	s := makeVersionSchema()
	diags := Diagnose([]string{"always", "bogus-key", "another-unknown"}, s, "")
	if len(diags) != 2 {
		t.Fatalf("expected 2 unknown diagnostics, got %d", len(diags))
	}
	for _, d := range diags {
		if d.Kind != "unknown" {
			t.Errorf("expected kind=unknown, got %q", d.Kind)
		}
	}
}

func TestDiagnose_Deprecated(t *testing.T) {
	s := makeVersionSchema()
	// deprecated field has Until=2.0.0, should trigger at v2.0.0+
	diags := Diagnose([]string{"deprecated"}, s, "2.0.0")
	if len(diags) != 1 {
		t.Fatalf("expected 1 deprecated diagnostic, got %d", len(diags))
	}
	if diags[0].Kind != "deprecated" {
		t.Errorf("expected kind=deprecated, got %q", diags[0].Kind)
	}
}

func TestDiagnose_DeprecatedNotYet(t *testing.T) {
	s := makeVersionSchema()
	// deprecated field has Until=2.0.0, should NOT trigger at v1.9.0
	diags := Diagnose([]string{"deprecated"}, s, "1.9.0")
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics at v1.9.0, got %d", len(diags))
	}
}

func TestDiagnose_Clean(t *testing.T) {
	s := makeVersionSchema()
	diags := Diagnose([]string{"always", "window"}, s, "2.0.0")
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for known non-deprecated keys, got %d", len(diags))
	}
}

func TestDiagnose_DuplicateKeys(t *testing.T) {
	s := makeVersionSchema()
	diags := Diagnose([]string{"bogus", "bogus", "bogus"}, s, "")
	if len(diags) != 1 {
		t.Errorf("duplicate config keys should produce 1 diagnostic, got %d", len(diags))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
