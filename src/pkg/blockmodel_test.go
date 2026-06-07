package pkg

import (
	"reflect"
	"strings"
	"testing"
)

// ssh-shaped openers + isNamed used across the acceptance cases: Host blocks are
// named unless the pattern is exactly "*"; every Match block is named.
var testOpeners = []string{"Host", "Match"}

func testIsNamed(opener, header string) bool {
	if strings.EqualFold(opener, "Match") {
		return true
	}
	return strings.TrimSpace(header) != "*"
}

func parseSSHish(data string) BlockModel {
	return Parse([]byte(data), testOpeners, testIsNamed)
}

func reconcileSSHish(data string, m BlockModel) []byte {
	return Reconcile([]byte(data), m, testOpeners, testIsNamed, PlacementRule{})
}

const sampleConfig = `# global defaults
ServerAliveInterval 60

# work box
Host work
    HostName work.example.com
    User alice
    IdentityFile ~/.ssh/work_ed25519
    IdentityFile ~/.ssh/work_rsa

Host db
    HostName db.internal
    Port 5432

Host *
    ForwardAgent no
`

// 1. Decode∘Encode identity.
func TestDecodeEncodeIdentity(t *testing.T) {
	m := parseSSHish(sampleConfig)
	if len(m.Blocks) == 0 {
		t.Fatal("expected parsed blocks")
	}
	got := Decode(Encode(m))
	if !reflect.DeepEqual(got, m) {
		t.Fatalf("Decode(Encode(m)) != m\n got: %#v\nwant: %#v", got, m)
	}
	// also Encode(Decode(s)) == s
	s := Encode(m)
	if Encode(Decode(s)) != s {
		t.Fatal("Encode(Decode(s)) != s")
	}
}

// helper: assert reconcile of an unedited parse equals the original bytes.
func assertNoOpStable(t *testing.T, data string) {
	t.Helper()
	out := reconcileSSHish(data, parseSSHish(data))
	if string(out) != data {
		t.Fatalf("no-op reconcile not byte-stable\n got: %q\nwant: %q", string(out), data)
	}
}

// 2 + 10. No-op byte-stable across LF, CRLF, and no-trailing-newline.
func TestNoOpByteStable(t *testing.T) {
	assertNoOpStable(t, sampleConfig)

	crlf := strings.ReplaceAll(sampleConfig, "\n", "\r\n")
	assertNoOpStable(t, crlf)

	noTrail := strings.TrimSuffix(sampleConfig, "\n")
	assertNoOpStable(t, noTrail)

	// crlf without trailing newline
	noTrailCRLF := strings.TrimSuffix(crlf, "\r\n")
	assertNoOpStable(t, noTrailCRLF)
}

// 3. Edit-one-directive isolation.
func TestEditOneDirectiveIsolation(t *testing.T) {
	m := parseSSHish(sampleConfig)

	// find block "work" and change its User to "bob".
	var edited bool
	for bi := range m.Blocks {
		if m.Blocks[bi].Header != "work" {
			continue
		}
		for ei := range m.Blocks[bi].Body {
			e := &m.Blocks[bi].Body[ei]
			if e.Kind == "directive" && e.Key == "User" {
				e.Values = []string{"bob"}
				edited = true
			}
		}
	}
	if !edited {
		t.Fatal("did not find User directive to edit")
	}

	out := string(reconcileSSHish(sampleConfig, m))
	if !strings.Contains(out, "    User bob\n") {
		t.Fatalf("expected edited line present, got:\n%s", out)
	}
	if strings.Contains(out, "User alice") {
		t.Fatalf("old value should be gone, got:\n%s", out)
	}

	// every other line byte-identical: diff line-by-line.
	wantLines := strings.Split(sampleConfig, "\n")
	gotLines := strings.Split(out, "\n")
	if len(wantLines) != len(gotLines) {
		t.Fatalf("line count changed: %d -> %d", len(wantLines), len(gotLines))
	}
	diffs := 0
	for i := range wantLines {
		if wantLines[i] != gotLines[i] {
			diffs++
			if !strings.Contains(gotLines[i], "User bob") {
				t.Fatalf("unexpected change at line %d: %q -> %q", i, wantLines[i], gotLines[i])
			}
		}
	}
	if diffs != 1 {
		t.Fatalf("expected exactly 1 changed line, got %d", diffs)
	}
}

// 4. Duplicate-pattern blocks survive as distinct blocks.
func TestDuplicatePatternBlocks(t *testing.T) {
	data := `Host foo
    HostName a.example.com

Host foo
    HostName b.example.com
`
	m := parseSSHish(data)
	if len(m.Blocks) != 2 {
		t.Fatalf("expected 2 distinct blocks, got %d", len(m.Blocks))
	}
	if m.Blocks[0].ID == m.Blocks[1].ID {
		t.Fatal("duplicate blocks must have distinct ids")
	}
	if m.Blocks[0].Header != "foo" || m.Blocks[1].Header != "foo" {
		t.Fatal("both headers should be foo")
	}
	// distinct content preserved
	if !blockHasDirective(m.Blocks[0], "HostName", "a.example.com") {
		t.Fatal("block 0 lost its HostName")
	}
	if !blockHasDirective(m.Blocks[1], "HostName", "b.example.com") {
		t.Fatal("block 1 lost its HostName")
	}
	assertNoOpStable(t, data)
}

// 5. Repeatable directives parse to N separate entries, each editable.
func TestRepeatableDirectives(t *testing.T) {
	data := `Host multi
    IdentityFile ~/.ssh/id_a
    IdentityFile ~/.ssh/id_b
    IdentityFile ~/.ssh/id_c
`
	m := parseSSHish(data)
	if len(m.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(m.Blocks))
	}
	var ids []*Entry
	for ei := range m.Blocks[0].Body {
		e := &m.Blocks[0].Body[ei]
		if e.Kind == "directive" && e.Key == "IdentityFile" {
			ids = append(ids, e)
		}
	}
	if len(ids) != 3 {
		t.Fatalf("expected 3 IdentityFile entries, got %d", len(ids))
	}
	assertNoOpStable(t, data)

	// edit only the middle one
	ids[1].Values = []string{"~/.ssh/id_NEW"}
	out := string(reconcileSSHish(data, m))
	if !strings.Contains(out, "IdentityFile ~/.ssh/id_NEW") {
		t.Fatalf("edited identity missing:\n%s", out)
	}
	if !strings.Contains(out, "IdentityFile ~/.ssh/id_a") || !strings.Contains(out, "IdentityFile ~/.ssh/id_c") {
		t.Fatalf("sibling identities should be untouched:\n%s", out)
	}
	if strings.Contains(out, "id_b") {
		t.Fatalf("old middle identity should be gone:\n%s", out)
	}
}

// 6. Reorder carries each block's owned leading comment/blank.
func TestReorderCarriesComment(t *testing.T) {
	data := `# first block comment
Host alpha
    HostName a

# second block comment
Host beta
    HostName b
`
	m := parseSSHish(data)
	if len(m.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(m.Blocks))
	}
	// swap order
	m.Blocks[0], m.Blocks[1] = m.Blocks[1], m.Blocks[0]

	out := string(Reconcile([]byte(data), m, testOpeners, testIsNamed, PlacementRule{}))

	idxBeta := strings.Index(out, "Host beta")
	idxAlpha := strings.Index(out, "Host alpha")
	if idxBeta < 0 || idxAlpha < 0 || idxBeta > idxAlpha {
		t.Fatalf("blocks not reordered:\n%s", out)
	}
	// each comment travels with its block: "second block comment" must precede
	// "Host beta", and "first block comment" must precede "Host alpha".
	idxSecond := strings.Index(out, "second block comment")
	idxFirst := strings.Index(out, "first block comment")
	if idxSecond >= idxBeta || idxBeta >= idxFirst || idxFirst >= idxAlpha {
		t.Fatalf("comments did not travel with their blocks:\n%s", out)
	}
}

// 7. Match-exec quoting preserved when editing elsewhere.
func TestMatchExecQuotingPreserved(t *testing.T) {
	data := `Match exec "ssh-proxy --check %h"
    ProxyJump bastion

Host plain
    User carol
`
	m := parseSSHish(data)
	if len(m.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(m.Blocks))
	}
	// the match block header must keep its quotes verbatim
	if m.Blocks[0].Opener != "Match" || m.Blocks[0].Header != `exec "ssh-proxy --check %h"` {
		t.Fatalf("match header not preserved: opener=%q header=%q", m.Blocks[0].Opener, m.Blocks[0].Header)
	}

	// edit the OTHER block's User
	for ei := range m.Blocks[1].Body {
		e := &m.Blocks[1].Body[ei]
		if e.Kind == "directive" && e.Key == "User" {
			e.Values = []string{"dave"}
		}
	}
	out := string(reconcileSSHish(data, m))
	if !strings.Contains(out, `Match exec "ssh-proxy --check %h"`) {
		t.Fatalf("match-exec quoting corrupted:\n%s", out)
	}
	if !strings.Contains(out, "User dave") {
		t.Fatalf("edit not applied:\n%s", out)
	}
}

// 8. Include lines preserved verbatim (root-level and between blocks).
func TestIncludeLinesPreserved(t *testing.T) {
	data := `Include ~/.ssh/config.d/*

Host one
    HostName one.example.com

Include ~/.ssh/more/*

Host two
    HostName two.example.com
`
	// Include is not an opener; it stays in flat regions. no-op must be stable.
	assertNoOpStable(t, data)

	// edit block "two" and confirm includes survive byte-identical
	m := parseSSHish(data)
	for bi := range m.Blocks {
		if m.Blocks[bi].Header != "two" {
			continue
		}
		for ei := range m.Blocks[bi].Body {
			e := &m.Blocks[bi].Body[ei]
			if e.Kind == "directive" && e.Key == "HostName" {
				e.Values = []string{"two.changed"}
			}
		}
	}
	out := string(reconcileSSHish(data, m))
	if !strings.Contains(out, "Include ~/.ssh/config.d/*\n") {
		t.Fatalf("root include lost:\n%s", out)
	}
	if !strings.Contains(out, "Include ~/.ssh/more/*\n") {
		t.Fatalf("inter-block include lost:\n%s", out)
	}
}

// 9. Precedence-correct insertion: a new named block inserts before a
// low-precedence flat block (Host *) per PlacementRule.
func TestPrecedenceCorrectInsertion(t *testing.T) {
	data := `Host existing
    HostName old.example.com

Host *
    ForwardAgent no
`
	m := parseSSHish(data)
	if len(m.Blocks) != 1 {
		t.Fatalf("expected 1 named block (Host * excluded), got %d", len(m.Blocks))
	}

	// build a brand-new named block and append it. appending changes order
	// relative to the recovered named ids, forcing the reorder path.
	newBlock := Block{
		ID:     "bNEW",
		Opener: "Host",
		Header: "fresh",
		Body: []Entry{
			{ID: "e0", Kind: "opener", Raw: []string{"Host fresh\n"}},
			{ID: "e1", Kind: "directive", Key: "HostName", Values: []string{"fresh.example.com"}, Raw: []string{"    HostName fresh.example.com\n"}},
		},
	}
	newBlock.RawSpan = nil // signal it is freshly rendered
	m.Blocks = append(m.Blocks, newBlock)

	place := PlacementRule{
		IsLowPrecedence: func(opener, header string) bool {
			return strings.EqualFold(opener, "Host") && strings.TrimSpace(header) == "*"
		},
	}
	out := string(Reconcile([]byte(data), m, testOpeners, testIsNamed, place))

	idxFresh := strings.Index(out, "Host fresh")
	idxWild := strings.Index(out, "Host *")
	if idxFresh < 0 {
		t.Fatalf("new block missing:\n%s", out)
	}
	if idxWild < 0 {
		t.Fatalf("Host * dropped:\n%s", out)
	}
	if idxFresh > idxWild {
		t.Fatalf("new block must precede Host * (first-match-wins):\n%s", out)
	}
	if !strings.Contains(out, "HostName fresh.example.com") {
		t.Fatalf("new block body missing:\n%s", out)
	}
}

// 11. A brand-new directive carries no Raw (the block editor builds it from
// Key+Values only); reconcile must rebuild the line from the block's inferred
// indent/eol/style instead of indexing an empty Raw. regression for a panic in
// inferIndent when adding the first directive to a freshly-created block on an
// empty config.
func TestNewDirectiveWithoutRaw(t *testing.T) {
	m := BlockModel{
		Blocks: []Block{{
			ID:      "b0",
			Opener:  "Host",
			Header:  "fresh",
			RawSpan: []string{"Host fresh\n"},
			Body: []Entry{
				{ID: "e0", Kind: "opener", Raw: []string{"Host fresh\n"}},
				// no Raw: mirrors editors.blockEditor.updateAddDirective
				{ID: "e1", Kind: "directive", Key: "HostName", Values: []string{"fresh.example.com"}},
			},
		}},
	}

	// empty current data forces the reorder path's firstNamed<0 branch, which is
	// where the panic surfaced.
	out := string(reconcileSSHish("", m))
	want := "Host fresh\n    HostName fresh.example.com\n"
	if out != want {
		t.Fatalf("new directive not rebuilt:\n got: %q\nwant: %q", out, want)
	}
}

func blockHasDirective(b Block, key, value string) bool {
	for _, e := range b.Body {
		if e.Kind == "directive" && e.Key == key && len(e.Values) == 1 && e.Values[0] == value {
			return true
		}
	}
	return false
}
