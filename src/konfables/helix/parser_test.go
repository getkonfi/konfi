package helix

import (
	"bytes"
	"os"
	"testing"

	"github.com/getkonfi/konfi/konfables"
	"github.com/getkonfi/konfi/pkg"
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
		{"theme", "gruvbox", true},
		{"editor.line-number", "relative", true},
		{"editor.mouse", "false", true},
		{"editor.cursor-shape.insert", "bar", true},
		{"editor.cursor-shape.normal", "block", true},
		{"keys.normal.C-s", ":w", true},
		{"nonexistent", "", false},
		{"editor.missing", "", false},
		{"editor.cursor-shape.missing", "", false},
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
		{"theme", 1, true},
		{"editor.line-number", 4, true},
		{"editor.cursor-shape.insert", 9, true},
		{"keys.normal.C-s", 13, true},
		{"nonexistent", -1, false},
		{"editor.missing", -1, false},
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
		{"replace section key", "editor.line-number", "\"absolute\"", "set_section.toml"},
		{"replace top-level key", "theme", "\"catppuccin\"", "set_toplevel.toml"},
		{"add to existing section", "editor.cursor-shape.select", "\"underline\"", "set_add_section.toml"},
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

func TestWriteFieldSerializesNumericListAsTOMLArray(t *testing.T) {
	p := newParser()
	data := []byte("[editor]\nrulers = [80]\n")
	field := pkg.Field{Key: "editor.rulers", Type: "list"}

	out, err := konfables.WriteField(p, data, field, "80\n120", "toml")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte(`rulers = [80, 120]`)) {
		t.Fatalf("list was not written as TOML numeric array:\n%s", out)
	}
}

func TestDeleteKey(t *testing.T) {
	p := newParser()

	tests := []struct {
		name   string
		key    string
		golden string
	}{
		{"delete section key", "editor.cursor-shape.normal", "delete_section.toml"},
		{"delete top-level key", "theme", "delete_toplevel.toml"},
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
		"theme",
		"editor.line-number", "editor.mouse", "editor.bufferline",
		"editor.cursor-shape.insert", "editor.cursor-shape.normal",
		"keys.normal.C-s",
	} {
		if !found[want] {
			t.Errorf("missing key %q in ListKeys output", want)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	p := newParser()

	src := []byte(`# helix editor config
theme = "gruvbox"

[editor]
line-number = "relative"
mouse = false
bufferline = "multiple"

[editor.cursor-shape]
insert = "bar"
normal = "block"

[keys.normal]
C-s = ":w"
`)

	// step 1: modify a section key
	out, err := p.SetValue(src, "editor.line-number", "\"absolute\"")
	if err != nil {
		t.Fatal(err)
	}
	v, ok := p.FindValue(out, "editor.line-number")
	if !ok || v != "absolute" {
		t.Fatalf("SetValue editor.line-number: got %q ok=%v", v, ok)
	}

	// step 2: modify a top-level key
	out, err = p.SetValue(out, "theme", "\"catppuccin\"")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "theme")
	if !ok || v != "catppuccin" {
		t.Fatalf("SetValue theme: got %q ok=%v", v, ok)
	}

	// step 3: add new key in nested section
	out, err = p.SetValue(out, "editor.cursor-shape.select", "\"underline\"")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "editor.cursor-shape.select")
	if !ok || v != "underline" {
		t.Fatalf("SetValue editor.cursor-shape.select: got %q ok=%v", v, ok)
	}

	// step 4: comments survived
	if !bytes.Contains(out, []byte("# helix editor config")) {
		t.Error("comment line lost during round-trip")
	}

	// step 5: untouched keys preserved
	for _, key := range []string{"editor.mouse", "editor.bufferline", "editor.cursor-shape.insert", "editor.cursor-shape.normal", "keys.normal.C-s"} {
		if _, ok := p.FindValue(out, key); !ok {
			t.Errorf("key %q lost during round-trip", key)
		}
	}

	// step 6: section headers preserved
	if !bytes.Contains(out, []byte("[editor]")) {
		t.Error("section header [editor] lost")
	}
	if !bytes.Contains(out, []byte("[editor.cursor-shape]")) {
		t.Error("section header [editor.cursor-shape] lost")
	}

	// step 7: ListKeys covers everything
	keys := p.ListKeys(out)
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	if !keySet["editor.cursor-shape.select"] {
		t.Error("ListKeys missing newly added editor.cursor-shape.select")
	}
}

func FuzzParser(f *testing.F) {
	f.Add([]byte("theme = \"gruvbox\"\n"), "theme")
	f.Add([]byte("[editor]\nline-number = \"relative\"\n"), "editor.line-number")
	f.Add([]byte("# comment\n\n[editor.cursor-shape]\ninsert = \"bar\"\n"), "editor.cursor-shape.insert")
	f.Add([]byte("[keys.normal]\nC-s = \":w\"\n"), "keys.normal.C-s")
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
