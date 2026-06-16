package sshd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/getkonfi/konfi/pkg"
)

var sampleConfig = []byte(`# server defaults
Port 22
PermitRootLogin no
PasswordAuthentication no

Match User deploy
    PasswordAuthentication yes
    ForceCommand internal-sftp
`)

func TestFindValueReadsOnlyGlobalDirectives(t *testing.T) {
	p := &parser{}

	got, ok := p.FindValue(sampleConfig, "PasswordAuthentication")
	if !ok || got != "no" {
		t.Fatalf("FindValue(global) = %q, %v; want no, true", got, ok)
	}
	if got, ok := p.FindValue(sampleConfig, "ForceCommand"); ok {
		t.Fatalf("FindValue should ignore Match body, got %q", got)
	}
}

func TestSetValueInsertsGlobalBeforeFirstMatch(t *testing.T) {
	p := &parser{}

	out, err := p.SetValue(sampleConfig, "ClientAliveInterval", "60")
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	setAt := strings.Index(text, "ClientAliveInterval 60")
	matchAt := strings.Index(text, "Match User deploy")
	if setAt < 0 {
		t.Fatalf("inserted directive missing:\n%s", text)
	}
	if matchAt < 0 || setAt > matchAt {
		t.Fatalf("global directive should be inserted before Match:\n%s", text)
	}
}

func TestBlocksExposeMatchBlocks(t *testing.T) {
	p := &parser{}

	enc, ok := p.FindValue(sampleConfig, "Blocks")
	if !ok {
		t.Fatal("Blocks should expose Match blocks")
	}
	model := pkg.Decode(enc)
	if len(model.Blocks) != 1 {
		t.Fatalf("blocks = %d, want 1", len(model.Blocks))
	}
	if model.Blocks[0].Opener != "Match" || model.Blocks[0].Header != "User deploy" {
		t.Fatalf("block = %q %q", model.Blocks[0].Opener, model.Blocks[0].Header)
	}
}

func TestDeleteBlocksKeepsGlobalDirectives(t *testing.T) {
	p := &parser{}

	out, err := p.DeleteKey(sampleConfig, "Blocks")
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if strings.Contains(text, "Match User deploy") {
		t.Fatalf("Match block was not removed:\n%s", text)
	}
	if !strings.Contains(text, "PermitRootLogin no") {
		t.Fatalf("global directives should remain:\n%s", text)
	}
}

func TestBlocksNoOpSetIsByteIdentical(t *testing.T) {
	p := &parser{}

	enc, ok := p.FindValue(sampleConfig, "Blocks")
	if !ok {
		t.Fatal("Blocks should exist")
	}
	out, err := p.SetValue(sampleConfig, "Blocks", enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, sampleConfig) {
		t.Fatalf("no-op SetValue(Blocks) changed bytes:\ngot:\n%s\nwant:\n%s", out, sampleConfig)
	}
}

func TestRepeatedGlobalPortDisplayMatchesEditedDirective(t *testing.T) {
	p := &parser{}
	data := []byte("Port 22\nPort 2222\n")

	all := p.FindAll(data)
	if got := all["Port"]; got != "22" {
		t.Fatalf("FindAll repeated Port = %q, want first value 22", got)
	}

	out, err := p.SetValue(data, "Port", "2200")
	if err != nil {
		t.Fatal(err)
	}
	all = p.FindAll(out)
	if got := all["Port"]; got != "2200" {
		t.Fatalf("FindAll after editing repeated Port = %q, want 2200:\n%s", got, out)
	}
	if !strings.Contains(string(out), "Port 2222") {
		t.Fatalf("second Port should remain untouched:\n%s", out)
	}
}

func TestRepeatedGlobalHostKeyDisplayMatchesEditedDirective(t *testing.T) {
	p := &parser{}
	data := []byte("HostKey /etc/ssh/ssh_host_ed25519_key\nHostKey /etc/ssh/ssh_host_rsa_key\n")

	all := p.FindAll(data)
	if got := all["HostKey"]; got != "/etc/ssh/ssh_host_ed25519_key" {
		t.Fatalf("FindAll repeated HostKey = %q, want first value", got)
	}

	out, err := p.SetValue(data, "HostKey", "/etc/ssh/ssh_host_ecdsa_key")
	if err != nil {
		t.Fatal(err)
	}
	all = p.FindAll(out)
	if got := all["HostKey"]; got != "/etc/ssh/ssh_host_ecdsa_key" {
		t.Fatalf("FindAll after editing repeated HostKey = %q, want edited first value:\n%s", got, out)
	}
	if !strings.Contains(string(out), "HostKey /etc/ssh/ssh_host_rsa_key") {
		t.Fatalf("second HostKey should remain untouched:\n%s", out)
	}
}
