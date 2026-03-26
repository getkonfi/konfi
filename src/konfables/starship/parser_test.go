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
	p := newParser()
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
	p := newParser()
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
	p := newParser()

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
	p := newParser()

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
	p := newParser()
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
	p := newParser()

	src := []byte(`# starship prompt config
format = "$all"
scan_timeout = 30

[character]
success_symbol = "[➜](bold green)"
error_symbol = "[✗](bold red)"

[git_branch]
style = "bold purple"
truncation_length = 8

[package]
disabled = true
`)

	// step 1: modify a section key
	// note: TOML helpers strip surrounding quotes on read
	out, err := p.SetValue(src, "character.success_symbol", "\"[→](bold cyan)\"")
	if err != nil {
		t.Fatal(err)
	}
	v, ok := p.FindValue(out, "character.success_symbol")
	if !ok || v != "[→](bold cyan)" {
		t.Fatalf("SetValue character.success_symbol: got %q ok=%v", v, ok)
	}

	// step 2: modify a top-level key
	out, err = p.SetValue(out, "scan_timeout", "60")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "scan_timeout")
	if !ok || v != "60" {
		t.Fatalf("SetValue scan_timeout: got %q ok=%v", v, ok)
	}

	// step 3: add a new key in existing section
	out, err = p.SetValue(out, "git_branch.format", "\"$branch\"")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "git_branch.format")
	if !ok || v != "$branch" {
		t.Fatalf("SetValue git_branch.format: got %q ok=%v", v, ok)
	}

	// step 4: verify comments survived
	if !bytes.Contains(out, []byte("# starship prompt config")) {
		t.Error("comment line lost during round-trip")
	}

	// step 5: verify untouched keys preserved
	for _, key := range []string{"format", "character.error_symbol", "git_branch.style", "package.disabled"} {
		if _, ok := p.FindValue(out, key); !ok {
			t.Errorf("key %q lost during round-trip", key)
		}
	}

	// step 6: verify section headers preserved
	if !bytes.Contains(out, []byte("[character]")) {
		t.Error("section header [character] lost")
	}
	if !bytes.Contains(out, []byte("[git_branch]")) {
		t.Error("section header [git_branch] lost")
	}

	// step 7: ListKeys covers everything
	keys := p.ListKeys(out)
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	if !keySet["git_branch.format"] {
		t.Error("ListKeys missing newly added git_branch.format")
	}
}

func FuzzParser(f *testing.F) {
	f.Add([]byte("format = \"$all\"\nscan_timeout = 30\n"), "format")
	f.Add([]byte("[character]\nsuccess_symbol = \"[➜](bold green)\"\n"), "character.success_symbol")
	f.Add([]byte("# comment\n\n[git_branch]\nstyle = \"bold purple\"\n"), "git_branch.style")
	f.Add([]byte("[package]\ndisabled = true\n"), "package.disabled")
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
