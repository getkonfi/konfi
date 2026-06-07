package gtk

import (
	"strings"
	"testing"
)

const testConfig = `[Settings]
gtk-theme-name=Adwaita-dark
gtk-icon-theme-name=Papirus-Dark
gtk-cursor-theme-name=Bibata-Modern-Classic
gtk-cursor-theme-size=24
gtk-font-name=JetBrainsMono Nerd Font 11
gtk-application-prefer-dark-theme=true
`

func TestFindValue(t *testing.T) {
	p := newParser()
	tests := []struct {
		key  string
		want string
		ok   bool
	}{
		{"Settings.gtk-theme-name", "Adwaita-dark", true},
		{"Settings.gtk-icon-theme-name", "Papirus-Dark", true},
		{"Settings.gtk-cursor-theme-size", "24", true},
		{"Settings.gtk-font-name", "JetBrainsMono Nerd Font 11", true},
		{"Settings.gtk-application-prefer-dark-theme", "true", true},
		{"Settings.missing", "", false},
		{"Other.key", "", false},
	}
	for _, tt := range tests {
		got, ok := p.FindValue([]byte(testConfig), tt.key)
		if ok != tt.ok || got != tt.want {
			t.Errorf("FindValue(%q) = %q, %v; want %q, %v", tt.key, got, ok, tt.want, tt.ok)
		}
	}
}

// replacing an existing key must preserve the `key=value` (no-space) line shape.
func TestSetValueReplacePreservesNoSpace(t *testing.T) {
	p := newParser()
	data, err := p.SetValue([]byte(testConfig), "Settings.gtk-theme-name", "Adwaita")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "gtk-theme-name=Adwaita\n") {
		t.Errorf("replaced line lost no-space form:\n%s", data)
	}
	if strings.Contains(string(data), "gtk-theme-name = ") {
		t.Errorf("replaced line gained spaces around =:\n%s", data)
	}
}

// inserting a new key must emit `key=value` with no spaces, matching GKeyFile.
func TestSetValueInsertNoSpace(t *testing.T) {
	p := newParser()
	data, err := p.SetValue([]byte(testConfig), "Settings.gtk-enable-animations", "false")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "gtk-enable-animations=false") {
		t.Errorf("inserted key not in no-space form:\n%s", data)
	}
	if strings.Contains(string(data), "gtk-enable-animations = ") {
		t.Errorf("inserted key gained spaces around =:\n%s", data)
	}
	got, ok := p.FindValue(data, "Settings.gtk-enable-animations")
	if !ok || got != "false" {
		t.Errorf("after insert FindValue = %q, %v; want false, true", got, ok)
	}
}

// inserting into a missing section creates `[Section]` then `key=value`.
func TestSetValueNewSectionNoSpace(t *testing.T) {
	p := newParser()
	data, err := p.SetValue([]byte(testConfig), "Other.foo", "bar")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "[Other]\nfoo=bar") {
		t.Errorf("new section not created in no-space form:\n%s", data)
	}
	got, ok := p.FindValue(data, "Other.foo")
	if !ok || got != "bar" {
		t.Errorf("after new section FindValue = %q, %v; want bar, true", got, ok)
	}
}

func TestDeleteKey(t *testing.T) {
	p := newParser()
	data, err := p.DeleteKey([]byte(testConfig), "Settings.gtk-cursor-theme-size")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.FindValue(data, "Settings.gtk-cursor-theme-size"); ok {
		t.Error("gtk-cursor-theme-size should be deleted")
	}
	got, ok := p.FindValue(data, "Settings.gtk-theme-name")
	if !ok || got != "Adwaita-dark" {
		t.Errorf("survivor gtk-theme-name = %q, %v; want Adwaita-dark, true", got, ok)
	}
}

func TestListKeys(t *testing.T) {
	p := newParser()
	keys := p.ListKeys([]byte(testConfig))
	expected := map[string]bool{
		"Settings.gtk-theme-name":                    true,
		"Settings.gtk-icon-theme-name":               true,
		"Settings.gtk-cursor-theme-name":             true,
		"Settings.gtk-cursor-theme-size":             true,
		"Settings.gtk-font-name":                     true,
		"Settings.gtk-application-prefer-dark-theme": true,
	}
	if len(keys) != len(expected) {
		t.Errorf("ListKeys: got %d keys, want %d (%v)", len(keys), len(expected), keys)
	}
	for _, k := range keys {
		if !expected[k] {
			t.Errorf("unexpected key: %q", k)
		}
	}
}

// full round-trip: read, replace, insert, delete, then verify line preservation
// and that every output line stays in `key=value` form.
func TestRoundTrip(t *testing.T) {
	p := newParser()
	data := []byte(testConfig)

	data, err := p.SetValue(data, "Settings.gtk-theme-name", "Adwaita")
	if err != nil {
		t.Fatal(err)
	}
	data, err = p.SetValue(data, "Settings.gtk-enable-animations", "false")
	if err != nil {
		t.Fatal(err)
	}
	data, err = p.DeleteKey(data, "Settings.gtk-icon-theme-name")
	if err != nil {
		t.Fatal(err)
	}

	if got, _ := p.FindValue(data, "Settings.gtk-theme-name"); got != "Adwaita" {
		t.Errorf("round-trip theme = %q", got)
	}
	if got, _ := p.FindValue(data, "Settings.gtk-enable-animations"); got != "false" {
		t.Errorf("round-trip animations = %q", got)
	}
	if _, ok := p.FindValue(data, "Settings.gtk-icon-theme-name"); ok {
		t.Error("round-trip: icon theme should be gone")
	}
	// untouched survivor
	if got, _ := p.FindValue(data, "Settings.gtk-font-name"); got != "JetBrainsMono Nerd Font 11" {
		t.Errorf("round-trip survivor font = %q", got)
	}

	// no assignment line may contain spaces around `=`
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed[0] == '[' {
			continue
		}
		if strings.Contains(line, " = ") {
			t.Errorf("line gained spaces around =: %q", line)
		}
	}
}

func TestFindAll(t *testing.T) {
	p := newParser()
	m := p.FindAll([]byte(testConfig))
	if len(m) != 6 {
		t.Errorf("FindAll: got %d entries, want 6", len(m))
	}
	if m["Settings.gtk-font-name"] != "JetBrainsMono Nerd Font 11" {
		t.Errorf("FindAll[gtk-font-name] = %q", m["Settings.gtk-font-name"])
	}
}

func FuzzParser(f *testing.F) {
	f.Add([]byte(testConfig), "Settings.gtk-theme-name")
	f.Add([]byte("[Settings]\ngtk-font-name=Sans 10\n"), "Settings.gtk-font-name")
	f.Add([]byte(""), "Settings.missing")
	f.Add([]byte("[Settings]\n"), "Settings.empty")

	p := newParser()
	f.Fuzz(func(t *testing.T, data []byte, key string) {
		p.FindValue(data, key)
		p.FindLine(data, key)
		p.ListKeys(data)
		if out, err := p.SetValue(data, key, "fuzzval"); err == nil {
			p.FindValue(out, key)
			p.ListKeys(out)
		}
		_, _ = p.DeleteKey(data, key)
	})
}
