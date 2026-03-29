package tmux

import (
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
	p := &parser{}
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
	p := &parser{}

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
	p := &parser{}
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
	p := &parser{}
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
