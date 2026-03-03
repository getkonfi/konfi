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

func TestRoundTrip(t *testing.T) {
	p := &parser{}

	src := []byte(`# alacritty configuration

[font]
size = 12.0

[font.normal]
family = "JetBrains Mono"

[colors.primary]
background = "#282828"
foreground = "#ebdbb2"

[window]
opacity = 1.0
decorations = "full"

[window.padding]
x = 8
y = 8
`)

	// step 1: modify a deeply nested value
	// note: TOML helpers strip surrounding quotes on read
	out, err := p.SetValue(src, "colors.primary.background", "\"#1e1e2e\"")
	if err != nil {
		t.Fatal(err)
	}
	v, ok := p.FindValue(out, "colors.primary.background")
	if !ok || v != "#1e1e2e" {
		t.Fatalf("SetValue colors.primary.background: got %q ok=%v", v, ok)
	}

	// step 2: modify a shallow value
	out, err = p.SetValue(out, "font.size", "14.0")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "font.size")
	if !ok || v != "14.0" {
		t.Fatalf("SetValue font.size: got %q ok=%v", v, ok)
	}

	// step 3: add a new key in existing section
	out, err = p.SetValue(out, "window.title", "\"Terminal\"")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "window.title")
	if !ok || v != "Terminal" {
		t.Fatalf("SetValue window.title: got %q ok=%v", v, ok)
	}

	// step 4: verify comments survived
	if !bytes.Contains(out, []byte("# alacritty configuration")) {
		t.Error("comment line lost during round-trip")
	}

	// step 5: verify untouched keys preserved
	for _, key := range []string{"font.normal.family", "colors.primary.foreground", "window.opacity", "window.padding.x", "window.padding.y"} {
		if _, ok := p.FindValue(out, key); !ok {
			t.Errorf("key %q lost during round-trip", key)
		}
	}

	// step 6: verify section headers preserved
	for _, hdr := range []string{"[font]", "[font.normal]", "[colors.primary]", "[window]", "[window.padding]"} {
		if !bytes.Contains(out, []byte(hdr)) {
			t.Errorf("section header %s lost", hdr)
		}
	}

	// step 7: ListKeys covers everything
	keys := p.ListKeys(out)
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	if !keySet["window.title"] {
		t.Error("ListKeys missing newly added window.title")
	}
	if !keySet["colors.primary.background"] {
		t.Error("ListKeys missing modified colors.primary.background")
	}
}

func FuzzParser(f *testing.F) {
	f.Add([]byte("[font]\nsize = 12.0\n"), "font.size")
	f.Add([]byte("[font.normal]\nfamily = \"JetBrains Mono\"\n"), "font.normal.family")
	f.Add([]byte("[colors.primary]\nbackground = \"#282828\"\n"), "colors.primary.background")
	f.Add([]byte("# comment\n\n[window]\nopacity = 1.0\n"), "window.opacity")
	f.Add([]byte(""), "missing")
	f.Add([]byte("[section]\n"), "section.key")
	f.Add([]byte("bare_key = true\n"), "bare_key")

	p := &parser{}
	f.Fuzz(func(t *testing.T, data []byte, key string) {
		p.FindValue(data, key)
		p.FindLine(data, key)
		p.ListKeys(data)
		if out, err := p.SetValue(data, key, "fuzzval"); err == nil {
			p.FindValue(out, key)
			p.ListKeys(out)
		}
		p.DeleteKey(data, key)
	})
}
