package rio

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
	p := newParser()
	data := loadTestdata(t, "config.toml")

	tests := []struct {
		key    string
		want   string
		wantOK bool
	}{
		{"padding-x", "8", true},
		{"renderer.performance", "High", true},
		{"renderer.backend", "Automatic", true},
		{"fonts.size", "18", true},
		{"fonts.regular.family", "JetBrains Mono", true},
		{"fonts.regular.style", "Normal", true},
		{"navigation.mode", "Plain", true},
		{"nonexistent", "", false},
		{"renderer.missing", "", false},
		{"fonts.regular.missing", "", false},
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
	p := newParser()
	data := loadTestdata(t, "config.toml")

	tests := []struct {
		key    string
		want   int
		wantOK bool
	}{
		{"padding-x", 1, true},
		{"renderer.performance", 4, true},
		{"fonts.size", 8, true},
		{"fonts.regular.family", 11, true},
		{"navigation.mode", 15, true},
		{"nonexistent", -1, false},
		{"renderer.missing", -1, false},
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
	p := newParser()

	tests := []struct {
		name   string
		key    string
		value  string
		golden string
	}{
		{"replace section key", "renderer.performance", "\"Low\"", "set_section.toml"},
		{"replace top-level key", "padding-x", "16", "set_toplevel.toml"},
		{"add to existing section", "fonts.regular.weight", "400", "set_add_section.toml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := loadTestdata(t, "config.toml")
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
	p := newParser()

	tests := []struct {
		name   string
		key    string
		golden string
	}{
		{"delete section key", "fonts.regular.style", "delete_section.toml"},
		{"delete top-level key", "padding-x", "delete_toplevel.toml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := loadTestdata(t, "config.toml")
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
	p := newParser()
	data := loadTestdata(t, "config.toml")

	got, err := p.DeleteKey(data, "nonexistent.key")
	if err != nil {
		t.Fatalf("DeleteKey(missing): %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Error("DeleteKey(missing) should return data unchanged")
	}
}

func TestListKeys(t *testing.T) {
	p := newParser()
	data := loadTestdata(t, "config.toml")
	keys := p.ListKeys(data)

	if len(keys) == 0 {
		t.Fatal("expected at least one key")
	}

	found := make(map[string]bool)
	for _, k := range keys {
		found[k] = true
	}
	for _, want := range []string{
		"padding-x",
		"renderer.performance", "renderer.backend",
		"fonts.size",
		"fonts.regular.family", "fonts.regular.style",
		"navigation.mode",
	} {
		if !found[want] {
			t.Errorf("missing key %q in ListKeys output", want)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	p := newParser()

	src := []byte(`# rio terminal config
padding-x = 8

[renderer]
performance = "High"
backend = "Automatic"

[fonts]
size = 18

[fonts.regular]
family = "JetBrains Mono"
style = "Normal"

[navigation]
mode = "Plain"
`)

	// step 1: modify a section key
	out, err := p.SetValue(src, "renderer.performance", "\"Low\"")
	if err != nil {
		t.Fatal(err)
	}
	v, ok := p.FindValue(out, "renderer.performance")
	if !ok || v != "Low" {
		t.Fatalf("SetValue renderer.performance: got %q ok=%v", v, ok)
	}

	// step 2: modify a top-level key
	out, err = p.SetValue(out, "padding-x", "16")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "padding-x")
	if !ok || v != "16" {
		t.Fatalf("SetValue padding-x: got %q ok=%v", v, ok)
	}

	// step 3: add new key in nested section
	out, err = p.SetValue(out, "fonts.regular.weight", "400")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "fonts.regular.weight")
	if !ok || v != "400" {
		t.Fatalf("SetValue fonts.regular.weight: got %q ok=%v", v, ok)
	}

	// step 4: comments survived
	if !bytes.Contains(out, []byte("# rio terminal config")) {
		t.Error("comment line lost during round-trip")
	}

	// step 5: untouched keys preserved
	for _, key := range []string{"renderer.backend", "fonts.size", "fonts.regular.family", "fonts.regular.style", "navigation.mode"} {
		if _, ok := p.FindValue(out, key); !ok {
			t.Errorf("key %q lost during round-trip", key)
		}
	}

	// step 6: section headers preserved
	if !bytes.Contains(out, []byte("[renderer]")) {
		t.Error("section header [renderer] lost")
	}
	if !bytes.Contains(out, []byte("[fonts.regular]")) {
		t.Error("section header [fonts.regular] lost")
	}

	// step 7: ListKeys covers everything
	keys := p.ListKeys(out)
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	if !keySet["fonts.regular.weight"] {
		t.Error("ListKeys missing newly added fonts.regular.weight")
	}
}

func FuzzParser(f *testing.F) {
	f.Add([]byte("padding-x = 8\n"), "padding-x")
	f.Add([]byte("[renderer]\nperformance = \"High\"\n"), "renderer.performance")
	f.Add([]byte("# comment\n\n[fonts.regular]\nfamily = \"JetBrains Mono\"\n"), "fonts.regular.family")
	f.Add([]byte("[navigation]\nmode = \"Plain\"\n"), "navigation.mode")
	f.Add([]byte(""), "missing")
	f.Add([]byte("[section]\n"), "section.key")

	p := newParser()
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
