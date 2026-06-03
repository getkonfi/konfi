package ssh

import (
	"testing"
)

func FuzzParser(f *testing.F) {
	f.Add([]byte("ServerAliveInterval 60\n"), "ServerAliveInterval")
	f.Add([]byte("Host *\n    AddKeysToAgent yes\n"), "AddKeysToAgent")
	f.Add([]byte("# comment\nCompression no\n"), "Compression")
	f.Add([]byte(""), "missing")
	f.Add([]byte("Host myserver\n    HostName example.com\n"), "HostName")
	f.Add([]byte("ForwardAgent = yes\n"), "ForwardAgent")

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

func TestRoundTrip(t *testing.T) {
	p := &parser{}
	data := []byte(testConfig)

	// replace global
	data, err := p.SetValue(data, "ServerAliveInterval", "120")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.FindValue(data, "ServerAliveInterval")
	if !ok || got != "120" {
		t.Fatalf("round-trip set global: got %q, %v", got, ok)
	}

	// replace in Host *
	data, err = p.SetValue(data, "Compression", "yes")
	if err != nil {
		t.Fatal(err)
	}
	got, ok = p.FindValue(data, "Compression")
	if !ok || got != "yes" {
		t.Fatalf("round-trip set Host *: got %q, %v", got, ok)
	}

	// delete
	data, err = p.DeleteKey(data, "AddKeysToAgent")
	if err != nil {
		t.Fatal(err)
	}
	_, ok = p.FindValue(data, "AddKeysToAgent")
	if ok {
		t.Fatal("round-trip delete: AddKeysToAgent should be gone")
	}

	// untouched survive
	got, ok = p.FindValue(data, "ServerAliveCountMax")
	if !ok || got != "3" {
		t.Fatalf("round-trip survival: got %q, %v", got, ok)
	}
}

func TestDeleteMissingKey(t *testing.T) {
	p := &parser{}
	out, err := p.DeleteKey([]byte(testConfig), "Nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(testConfig) {
		t.Error("deleting missing key should be no-op")
	}
}

func TestFindAll(t *testing.T) {
	p := &parser{}
	m := p.FindAll([]byte(testConfig))
	if len(m) != 6 {
		t.Errorf("FindAll: got %d entries %v, want 6", len(m), m)
	}
	if m["ServerAliveInterval"] != "60" {
		t.Errorf("FindAll[ServerAliveInterval] = %q", m["ServerAliveInterval"])
	}
}

func TestEqualsSeparator(t *testing.T) {
	p := &parser{}
	data := []byte("ForwardAgent = yes\nServerAliveInterval = 60\n")
	val, ok := p.FindValue(data, "ForwardAgent")
	if !ok || val != "yes" {
		t.Errorf("FindValue with = separator: got %q, %v", val, ok)
	}
	val, ok = p.FindValue(data, "ServerAliveInterval")
	if !ok || val != "60" {
		t.Errorf("FindValue with = separator: got %q, %v", val, ok)
	}
}

func TestInsertCreatesHostWildcard(t *testing.T) {
	p := &parser{}
	data := []byte("ServerAliveInterval 60\n")
	out, err := p.SetValue(data, "ForwardAgent", "yes")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.FindValue(out, "ForwardAgent")
	if !ok || got != "yes" {
		t.Errorf("insert new key: got %q, %v", got, ok)
	}
}

func TestMatchBlockExcluded(t *testing.T) {
	p := &parser{}
	// SSH config convention: global settings come first, then Host/Match blocks.
	// scope only resets on Host/Match lines, so global keys after a Match block
	// would be excluded. This test verifies correct scoping.
	data := []byte("ServerAliveInterval 60\n\nMatch host foo\n    User bar\n")
	val, ok := p.FindValue(data, "User")
	if ok {
		t.Errorf("keys in Match block should be excluded, got %q", val)
	}
	val, ok = p.FindValue(data, "ServerAliveInterval")
	if !ok || val != "60" {
		t.Errorf("global key should be found: got %q, %v", val, ok)
	}
}
