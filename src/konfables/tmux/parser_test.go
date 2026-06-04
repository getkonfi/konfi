package tmux

import (
	"bytes"
	"testing"
)

const testConfig = `# tmux config
set -g default-terminal "tmux-256color"
set -g escape-time 0
set-option -g mouse on
set -g history-limit 10000
set -g prefix C-a
`

func TestFindValue(t *testing.T) {
	p := newParser()
	tests := []struct {
		key  string
		want string
		ok   bool
	}{
		{"default-terminal", `"tmux-256color"`, true},
		{"escape-time", "0", true},
		{"mouse", "on", true},
		{"history-limit", "10000", true},
		{"prefix", "C-a", true},
		{"missing", "", false},
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
	data, err := p.SetValue([]byte(testConfig), "escape-time", "10")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.FindValue(data, "escape-time")
	if !ok || got != "10" {
		t.Errorf("after SetValue: got %q, %v; want 10, true", got, ok)
	}

	// insert new
	data, err = p.SetValue([]byte(testConfig), "base-index", "1")
	if err != nil {
		t.Fatal(err)
	}
	got, ok = p.FindValue(data, "base-index")
	if !ok || got != "1" {
		t.Errorf("after SetValue new: got %q, %v; want 1, true", got, ok)
	}
}

func TestDeleteKey(t *testing.T) {
	p := newParser()
	data, err := p.DeleteKey([]byte(testConfig), "mouse")
	if err != nil {
		t.Fatal(err)
	}
	_, ok := p.FindValue(data, "mouse")
	if ok {
		t.Error("mouse should be deleted")
	}
	// other keys should remain
	got, ok := p.FindValue(data, "escape-time")
	if !ok || got != "0" {
		t.Errorf("escape-time should still exist: got %q, %v", got, ok)
	}
}

func TestListKeys(t *testing.T) {
	p := newParser()
	keys := p.ListKeys([]byte(testConfig))
	expected := map[string]bool{
		"default-terminal": true,
		"escape-time":      true,
		"mouse":            true,
		"history-limit":    true,
		"prefix":           true,
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
	f.Add([]byte("set -g default-terminal \"tmux-256color\"\n"), "default-terminal")
	f.Add([]byte("set -g escape-time 0\nset-option -g mouse on\n"), "escape-time")
	f.Add([]byte("# comment\nset -g history-limit 10000\n\n"), "history-limit")
	f.Add([]byte(""), "missing")
	f.Add([]byte("not-a-set-line\n"), "key")
	f.Add([]byte("setw -g mode-keys vi\n"), "mode-keys")

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
	data, err := p.SetValue(data, "escape-time", "50")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.FindValue(data, "escape-time")
	if !ok || got != "50" {
		t.Fatalf("round-trip set: got %q, %v", got, ok)
	}

	// add new
	data, err = p.SetValue(data, "base-index", "1")
	if err != nil {
		t.Fatal(err)
	}
	got, ok = p.FindValue(data, "base-index")
	if !ok || got != "1" {
		t.Fatalf("round-trip add: got %q, %v", got, ok)
	}

	// delete
	data, err = p.DeleteKey(data, "prefix")
	if err != nil {
		t.Fatal(err)
	}
	_, ok = p.FindValue(data, "prefix")
	if ok {
		t.Fatal("round-trip delete: prefix should be gone")
	}

	// untouched survive
	got, ok = p.FindValue(data, "mouse")
	if !ok || got != "on" {
		t.Fatalf("round-trip survival: got %q, %v", got, ok)
	}
}

func TestDeleteMissingKey(t *testing.T) {
	p := newParser()
	data := []byte(testConfig)
	out, err := p.DeleteKey(data, "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, data) {
		t.Error("deleting missing key should be no-op")
	}
}

func TestFindAll(t *testing.T) {
	p := newParser()
	m := p.FindAll([]byte(testConfig))
	if len(m) != 5 {
		t.Errorf("FindAll: got %d entries, want 5", len(m))
	}
	if m["mouse"] != "on" {
		t.Errorf("FindAll[mouse] = %q", m["mouse"])
	}
}
