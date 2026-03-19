package git

import (
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
	p := &parser{}
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
	p := &parser{}

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
	p := &parser{}
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
	p := &parser{}
	keys := p.ListKeys([]byte(testConfig))
	expected := map[string]bool{
		"user.name":         true,
		"user.email":        true,
		"core.editor":       true,
		"core.autocrlf":     true,
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
