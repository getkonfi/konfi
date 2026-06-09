package git

import (
	"strings"
	"testing"
)

const testConfig = `[user]
	name = John Doe
	email = john@example.com
[core]
	editor = vim
	autocrlf = input
[init]
	defaultBranch = main
`

func TestFindValue(t *testing.T) {
	p := newParser()
	tests := []struct {
		key  string
		want string
		ok   bool
	}{
		{"user.name", "John Doe", true},
		{"user.email", "john@example.com", true},
		{"core.editor", "vim", true},
		{"core.autocrlf", "input", true},
		{"init.defaultBranch", "main", true},
		{"user.missing", "", false},
		{"missing.key", "", false},
	}
	for _, tt := range tests {
		got, ok := p.FindValue([]byte(testConfig), tt.key)
		if ok != tt.ok || got != tt.want {
			t.Errorf("FindValue(%q) = %q, %v; want %q, %v", tt.key, got, ok, tt.want, tt.ok)
		}
	}
}

func TestSetValue(t *testing.T) {
	p := newParser()

	// replace existing
	data, err := p.SetValue([]byte(testConfig), "user.name", "Jane Doe")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.FindValue(data, "user.name")
	if !ok || got != "Jane Doe" {
		t.Errorf("after SetValue: got %q, %v; want Jane Doe, true", got, ok)
	}

	// insert new key in existing section
	data, err = p.SetValue([]byte(testConfig), "core.pager", "less")
	if err != nil {
		t.Fatal(err)
	}
	got, ok = p.FindValue(data, "core.pager")
	if !ok || got != "less" {
		t.Errorf("after SetValue new key: got %q, %v; want less, true", got, ok)
	}

	// insert new section
	data, err = p.SetValue([]byte(testConfig), "push.default", "simple")
	if err != nil {
		t.Fatal(err)
	}
	got, ok = p.FindValue(data, "push.default")
	if !ok || got != "simple" {
		t.Errorf("after SetValue new section: got %q, %v; want simple, true", got, ok)
	}
}

func TestDeleteKey(t *testing.T) {
	p := newParser()
	data, err := p.DeleteKey([]byte(testConfig), "core.editor")
	if err != nil {
		t.Fatal(err)
	}
	_, ok := p.FindValue(data, "core.editor")
	if ok {
		t.Error("core.editor should be deleted")
	}
	// other keys should remain
	got, ok := p.FindValue(data, "core.autocrlf")
	if !ok || got != "input" {
		t.Errorf("core.autocrlf should still exist: got %q, %v", got, ok)
	}
}

func TestListKeys(t *testing.T) {
	p := newParser()
	keys := p.ListKeys([]byte(testConfig))
	expected := map[string]bool{
		"user.name":          true,
		"user.email":         true,
		"core.editor":        true,
		"core.autocrlf":      true,
		"init.defaultBranch": true,
	}
	if len(keys) != len(expected) {
		t.Errorf("ListKeys: got %d keys, want %d", len(keys), len(expected))
	}
	for _, k := range keys {
		if !expected[k] {
			t.Errorf("unexpected key: %q", k)
		}
	}
}

func FuzzParser(f *testing.F) {
	f.Add([]byte("[user]\n\tname = John\n"), "user.name")
	f.Add([]byte("[core]\n\teditor = vim\n\tautocrlf = input\n"), "core.editor")
	f.Add([]byte("# comment\n[remote \"origin\"]\n\turl = git@github.com:user/repo.git\n"), "remote.origin.url")
	f.Add([]byte(""), "missing")
	f.Add([]byte("[empty]\n"), "empty.key")
	f.Add([]byte("; semicolon comment\n[alias]\n\tst = status\n"), "alias.st")

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

func TestRoundTrip(t *testing.T) {
	p := newParser()
	data := []byte(testConfig)

	// replace existing
	data, err := p.SetValue(data, "user.name", "Jane Doe")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.FindValue(data, "user.name")
	if !ok || got != "Jane Doe" {
		t.Fatalf("round-trip set: got %q, %v", got, ok)
	}

	// add to existing section
	data, err = p.SetValue(data, "user.signingkey", "ABC123")
	if err != nil {
		t.Fatal(err)
	}
	got, ok = p.FindValue(data, "user.signingkey")
	if !ok || got != "ABC123" {
		t.Fatalf("round-trip add: got %q, %v", got, ok)
	}

	// delete
	data, err = p.DeleteKey(data, "core.editor")
	if err != nil {
		t.Fatal(err)
	}
	_, ok = p.FindValue(data, "core.editor")
	if ok {
		t.Fatal("round-trip delete: core.editor should be gone")
	}

	// untouched survive
	got, ok = p.FindValue(data, "core.autocrlf")
	if !ok || got != "input" {
		t.Fatalf("round-trip survival: got %q, %v", got, ok)
	}
}

func TestDeleteMissingKey(t *testing.T) {
	p := newParser()
	out, err := p.DeleteKey([]byte(testConfig), "missing.key")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(testConfig) {
		t.Error("deleting missing key should be no-op")
	}
}

func TestFindAll(t *testing.T) {
	p := newParser()
	m := p.FindAll([]byte(testConfig))
	if len(m) != 5 {
		t.Errorf("FindAll: got %d entries, want 5", len(m))
	}
	if m["user.name"] != "John Doe" {
		t.Errorf("FindAll[user.name] = %q", m["user.name"])
	}
}

func TestSemicolonComment(t *testing.T) {
	p := newParser()
	data := []byte("; this is a comment\n[user]\n\tname = Test\n")
	keys := p.ListKeys(data)
	if len(keys) != 1 || keys[0] != "user.name" {
		t.Errorf("ListKeys with ; comment = %v", keys)
	}
}

func TestColorDiffCanonicalSubsection(t *testing.T) {
	p := newParser()
	data := []byte("[color \"diff\"]\n\tmeta = yellow bold\n\tfrag = magenta bold\n")

	got, ok := p.FindValue(data, "color.diff.meta")
	if !ok || got != "yellow bold" {
		t.Fatalf("FindValue(color.diff.meta) = %q, %v; want yellow bold, true", got, ok)
	}
	got, ok = p.FindValue(data, "color.diff.frag")
	if !ok || got != "magenta bold" {
		t.Fatalf("FindValue(color.diff.frag) = %q, %v; want magenta bold, true", got, ok)
	}

	all := p.FindAll(data)
	if all["color.diff.meta"] != "yellow bold" {
		t.Fatalf("FindAll[color.diff.meta] = %q", all["color.diff.meta"])
	}
}

func TestSetColorDiffWritesCanonicalSubsection(t *testing.T) {
	p := newParser()

	out, err := p.SetValue([]byte(testConfig), "color.diff.meta", "yellow bold")
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "[color \"diff\"]") || !strings.Contains(text, "meta = yellow bold") {
		t.Fatalf("canonical color diff setting missing:\n%s", text)
	}
	if strings.Contains(text, "diff.meta =") {
		t.Fatalf("wrote invalid [color] diff.meta setting:\n%s", text)
	}
}

func TestSetColorDiffConvertsLegacyInvalidKey(t *testing.T) {
	p := newParser()
	data := []byte("[color]\n\tdiff.meta = blue\n")

	got, ok := p.FindValue(data, "color.diff.meta")
	if !ok || got != "blue" {
		t.Fatalf("legacy FindValue(color.diff.meta) = %q, %v; want blue, true", got, ok)
	}

	out, err := p.SetValue(data, "color.diff.meta", "yellow")
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if strings.Contains(text, "diff.meta =") {
		t.Fatalf("legacy invalid key should be removed:\n%s", text)
	}
	if !strings.Contains(text, "[color \"diff\"]") || !strings.Contains(text, "meta = yellow") {
		t.Fatalf("canonical replacement missing:\n%s", text)
	}
}
