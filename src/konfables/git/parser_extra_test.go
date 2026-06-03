package git

import (
	"testing"
)

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
