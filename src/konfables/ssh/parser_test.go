package ssh

import (
	"testing"
)

const testConfig = `# global settings
ServerAliveInterval 60
ServerAliveCountMax 3

Host *
    AddKeysToAgent yes
    Compression no
    IdentityFile ~/.ssh/id_ed25519

Host myserver
    HostName example.com
    User admin
    Port 2222
`

func TestFindValue(t *testing.T) {
	p := &parser{}
	tests := []struct {
		key  string
		want string
		ok   bool
	}{
		{"ServerAliveInterval", "60", true},
		{"ServerAliveCountMax", "3", true},
		{"AddKeysToAgent", "yes", true},
		{"Compression", "no", true},
		{"IdentityFile", "~/.ssh/id_ed25519", true},
		// should NOT find keys in specific Host blocks
		{"HostName", "", false},
		{"Port", "", false},
		{"Missing", "", false},
	}
	for _, tt := range tests {
		got, ok := p.FindValue([]byte(testConfig), tt.key)
		if ok != tt.ok || got != tt.want {
			t.Errorf("FindValue(%q) = %q, %v; want %q, %v", tt.key, got, ok, tt.want, tt.ok)
		}
	}
}

func TestFindValueCaseInsensitive(t *testing.T) {
	p := &parser{}
	got, ok := p.FindValue([]byte(testConfig), "serveraliveinterval")
	if !ok || got != "60" {
		t.Errorf("case-insensitive FindValue: got %q, %v; want 60, true", got, ok)
	}
}

func TestSetValue(t *testing.T) {
	p := &parser{}

	// replace existing
	data, err := p.SetValue([]byte(testConfig), "ServerAliveInterval", "120")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.FindValue(data, "ServerAliveInterval")
	if !ok || got != "120" {
		t.Errorf("after SetValue: got %q, %v; want 120, true", got, ok)
	}

	// replace in Host * block
	data, err = p.SetValue([]byte(testConfig), "AddKeysToAgent", "no")
	if err != nil {
		t.Fatal(err)
	}
	got, ok = p.FindValue(data, "AddKeysToAgent")
	if !ok || got != "no" {
		t.Errorf("after SetValue Host *: got %q, %v; want no, true", got, ok)
	}

	// insert new key
	data, err = p.SetValue([]byte(testConfig), "ForwardAgent", "yes")
	if err != nil {
		t.Fatal(err)
	}
	got, ok = p.FindValue(data, "ForwardAgent")
	if !ok || got != "yes" {
		t.Errorf("after SetValue new: got %q, %v; want yes, true", got, ok)
	}
}

func TestDeleteKey(t *testing.T) {
	p := &parser{}
	data, err := p.DeleteKey([]byte(testConfig), "Compression")
	if err != nil {
		t.Fatal(err)
	}
	_, ok := p.FindValue(data, "Compression")
	if ok {
		t.Error("Compression should be deleted")
	}
	// other keys should remain
	got, ok := p.FindValue(data, "AddKeysToAgent")
	if !ok || got != "yes" {
		t.Errorf("AddKeysToAgent should still exist: got %q, %v", got, ok)
	}
}

func TestListKeys(t *testing.T) {
	p := &parser{}
	keys := p.ListKeys([]byte(testConfig))
	// should include global and Host * keys, but NOT Host-specific ones
	expected := map[string]bool{
		"ServerAliveInterval": true,
		"ServerAliveCountMax": true,
		"AddKeysToAgent":      true,
		"Compression":         true,
		"IdentityFile":        true,
	}
	if len(keys) != len(expected) {
		t.Errorf("ListKeys: got %d keys %v, want %d", len(keys), keys, len(expected))
	}
	for _, k := range keys {
		if !expected[k] {
			t.Errorf("unexpected key: %q", k)
		}
	}
}
