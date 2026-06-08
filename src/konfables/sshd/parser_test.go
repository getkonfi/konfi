package sshd

import (
	"strings"
	"testing"

	"github.com/eminert/konfi/pkg"
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
	if string(out) != string(sampleConfig) {
		t.Fatalf("no-op SetValue(Blocks) changed bytes:\ngot:\n%s\nwant:\n%s", out, sampleConfig)
	}
}
