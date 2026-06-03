package tmux

import (
	"bytes"
	"testing"
)

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
