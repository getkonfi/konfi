package ssh

import (
	"strings"
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
	// should include global and Host * keys plus synthetic Hosts.
	expected := map[string]bool{
		"ServerAliveInterval": true,
		"ServerAliveCountMax": true,
		"AddKeysToAgent":      true,
		"Compression":         true,
		"IdentityFile":        true,
		"Hosts":               true,
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

func TestFindHostsValue(t *testing.T) {
	p := &parser{}
	got, ok := p.FindValue([]byte(testConfig), "Hosts")
	want := "myserver | example.com | admin | 2222 |  | "
	if !ok || got != want {
		t.Errorf("FindValue(Hosts) = %q, %v; want %q, true", got, ok, want)
	}
}

func TestSetHostsValuePreservesUnknownDirectives(t *testing.T) {
	p := &parser{}
	input := []byte(`# global
ServerAliveInterval 60

Host dev
    HostName old.example.com
    User olduser
    LocalForward 8080 localhost:8080

Host *
    AddKeysToAgent yes
`)

	value := "dev | new.example.com | admin | 2222 | ~/.ssh/dev_ed25519 | bastion\n" +
		"github.com | github.com | git |  | ~/.ssh/gh_ed25519 | "
	out, err := p.SetValue(input, "Hosts", value)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)

	for _, want := range []string{
		"Host dev",
		"    LocalForward 8080 localhost:8080",
		"    HostName new.example.com",
		"    User admin",
		"    Port 2222",
		"    IdentityFile ~/.ssh/dev_ed25519",
		"    ProxyJump bastion",
		"Host github.com",
		"    HostName github.com",
		"    User git",
		"Host *",
		"    AddKeysToAgent yes",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("updated config missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "old.example.com") || strings.Contains(text, "olduser") {
		t.Errorf("old host directives were not replaced:\n%s", text)
	}
	if strings.Index(text, "Host github.com") > strings.Index(text, "Host *") {
		t.Errorf("new host block should be inserted before Host * defaults:\n%s", text)
	}
}

func TestSetHostsValueFromEmptyConfig(t *testing.T) {
	p := &parser{}
	out, err := p.SetValue(nil, "Hosts", "dev | dev.example.com | admin |  |  | ")
	if err != nil {
		t.Fatal(err)
	}
	want := "Host dev\n    HostName dev.example.com\n    User admin"
	if string(out) != want {
		t.Errorf("SetValue from empty config:\ngot:\n%s\nwant:\n%s", out, want)
	}
}

func TestDeleteHostsValue(t *testing.T) {
	p := &parser{}
	out, err := p.DeleteKey([]byte(testConfig), "Hosts")
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if strings.Contains(text, "Host myserver") {
		t.Errorf("specific host block should be deleted:\n%s", text)
	}
	if !strings.Contains(text, "Host *") || !strings.Contains(text, "AddKeysToAgent yes") {
		t.Errorf("Host * defaults should remain:\n%s", text)
	}
}

func TestFindAllCanonicalizesKeys(t *testing.T) {
	p := &parser{}
	data := []byte("hostname example.org\nHost *\n    addkeystoagent yes\n")
	all := p.FindAll(data)
	if got := all["HostName"]; got != "example.org" {
		t.Errorf("FindAll HostName = %q, want example.org", got)
	}
	if got := all["AddKeysToAgent"]; got != "yes" {
		t.Errorf("FindAll AddKeysToAgent = %q, want yes", got)
	}
}
