package ssh

import (
	"bytes"
	"strings"
	"testing"

	"github.com/eminert/konfi/konfables"
	"github.com/eminert/konfi/pkg"
)

// blocksParser returns a fully-configured parser (palette + engine predicates)
// for exercising the synthetic "Blocks" key.
func blocksParser() konfables.Parser {
	return New(pkg.NewFilePersister("")).Parser()
}

// configuredParser returns the concrete parser with engine predicates wired, for
// tests that need methods beyond the konfables.Parser interface (e.g. FindAll).
func configuredParser() *parser {
	return New(pkg.NewFilePersister("")).Parser().(*parser)
}

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
	p := blocksParser()
	keys := p.ListKeys([]byte(testConfig))
	// should include global and Host * keys plus synthetic Blocks.
	expected := map[string]bool{
		"ServerAliveInterval": true,
		"ServerAliveCountMax": true,
		"AddKeysToAgent":      true,
		"Compression":         true,
		"IdentityFile":        true,
		"Blocks":              true,
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

func TestBlocksFindExposesNamedHost(t *testing.T) {
	p := blocksParser()
	enc, ok := p.FindValue([]byte(testConfig), "Blocks")
	if !ok {
		t.Fatal("FindValue(Blocks) should expose the named Host myserver block")
	}
	model := pkg.Decode(enc)
	if len(model.Blocks) != 1 {
		t.Fatalf("expected 1 named block (Host myserver), got %d", len(model.Blocks))
	}
	b := model.Blocks[0]
	if b.Opener != "Host" || b.Header != "myserver" {
		t.Errorf("named block header not preserved: opener=%q header=%q", b.Opener, b.Header)
	}
	for _, want := range []string{"HostName", "User", "Port"} {
		found := false
		for _, e := range b.Body {
			if e.Kind == "directive" && e.Key == want {
				found = true
			}
		}
		if !found {
			t.Errorf("named block missing directive %q", want)
		}
	}
}

func TestBlocksDeleteRemovesNamedHost(t *testing.T) {
	p := blocksParser()
	out, err := p.DeleteKey([]byte(testConfig), "Blocks")
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if strings.Contains(text, "Host myserver") {
		t.Errorf("named host block should be deleted:\n%s", text)
	}
	if !strings.Contains(text, "Host *") || !strings.Contains(text, "AddKeysToAgent yes") {
		t.Errorf("Host * defaults should remain:\n%s", text)
	}
}

func TestFindAllCanonicalizesKeys(t *testing.T) {
	p := configuredParser()
	data := []byte("hostname example.org\nHost *\n    addkeystoagent yes\n")
	all := p.FindAll(data)
	if got := all["HostName"]; got != "example.org" {
		t.Errorf("FindAll HostName = %q, want example.org", got)
	}
	if got := all["AddKeysToAgent"]; got != "yes" {
		t.Errorf("FindAll AddKeysToAgent = %q, want yes", got)
	}
}

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
	p := configuredParser()
	m := p.FindAll([]byte(testConfig))
	// 5 flat directives (global + Host *) plus the synthetic Blocks entry.
	if len(m) != 6 {
		t.Errorf("FindAll: got %d entries %v, want 6", len(m), m)
	}
	if m["ServerAliveInterval"] != "60" {
		t.Errorf("FindAll[ServerAliveInterval] = %q", m["ServerAliveInterval"])
	}
	if _, ok := m["Blocks"]; !ok {
		t.Errorf("FindAll should include synthetic Blocks entry: %v", m)
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

// --- synthetic "Blocks" key (engine-backed, named blocks only) ---

func TestPaletteExcludesSyntheticFields(t *testing.T) {
	pp, ok := blocksParser().(konfables.PaletteProvider)
	if !ok {
		t.Fatal("parser does not implement PaletteProvider")
	}
	palette := pp.Palette()
	if len(palette) == 0 {
		t.Fatal("palette is empty")
	}
	for _, f := range palette {
		if isBlocksKey(f.Key) {
			t.Errorf("palette must not contain synthetic field %q", f.Key)
		}
	}
	// spot-check a known flat directive carries its types/bounds.
	var port *pkg.Field
	for i := range palette {
		if palette[i].Key == "Port" {
			port = &palette[i]
		}
	}
	if port == nil {
		t.Fatal("expected Port in palette")
	}
	if port.Type != "number" || port.Min == nil || port.Max == nil {
		t.Errorf("Port field missing type/bounds: %+v", port)
	}
}

func TestBlocksFindEmptyWhenNoNamedBlocks(t *testing.T) {
	p := blocksParser()
	// only root + default Host * — no named blocks.
	data := []byte("ServerAliveInterval 60\n\nHost *\n    AddKeysToAgent yes\n")
	if _, ok := p.FindValue(data, "Blocks"); ok {
		t.Error("Blocks should report ok=false when there are no named blocks")
	}
}

func TestBlocksMatchModeledAsNamed(t *testing.T) {
	p := blocksParser()
	data := []byte(`Host *
    AddKeysToAgent yes

Match exec "test -f /tmp/flag"
    ForwardAgent yes
    User scoped
`)
	enc, ok := p.FindValue(data, "Blocks")
	if !ok {
		t.Fatal("Match block should be exposed as a named block")
	}
	model := pkg.Decode(enc)
	if len(model.Blocks) != 1 {
		t.Fatalf("expected 1 named block, got %d", len(model.Blocks))
	}
	b := model.Blocks[0]
	if b.Opener != "Match" || b.Header != `exec "test -f /tmp/flag"` {
		t.Errorf("match header not preserved: opener=%q header=%q", b.Opener, b.Header)
	}

	// edit a directive elsewhere (the Match body) and verify quoting in the
	// header survives the round-trip.
	for i := range model.Blocks[0].Body {
		e := &model.Blocks[0].Body[i]
		if e.Kind == "directive" && e.Key == "User" {
			e.Values = []string{"edited"}
		}
	}
	out, err := p.SetValue(data, "Blocks", pkg.Encode(model))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, `Match exec "test -f /tmp/flag"`) {
		t.Errorf("match header quoting lost:\n%s", text)
	}
	if !strings.Contains(text, "    User edited") {
		t.Errorf("directive edit not applied:\n%s", text)
	}
	if !strings.Contains(text, "    ForwardAgent yes") {
		t.Errorf("sibling directive should survive:\n%s", text)
	}
}

func TestBlocksDuplicateHostsSurvive(t *testing.T) {
	p := blocksParser()
	data := []byte(`Host foo
    HostName a.example.com

Host foo
    HostName b.example.com
`)
	enc, ok := p.FindValue(data, "Blocks")
	if !ok {
		t.Fatal("expected named blocks")
	}
	model := pkg.Decode(enc)
	if len(model.Blocks) != 2 {
		t.Fatalf("duplicate Host foo should be two blocks, got %d", len(model.Blocks))
	}
	out, err := p.SetValue(data, "Blocks", enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, data) {
		t.Errorf("no-op round-trip of duplicate hosts not byte-identical:\ngot:\n%s\nwant:\n%s", out, data)
	}
}

func TestBlocksMultipleIdentityFilesPreserved(t *testing.T) {
	p := blocksParser()
	data := []byte(`Host gw
    IdentityFile ~/.ssh/a
    IdentityFile ~/.ssh/b
    IdentityFile ~/.ssh/c
`)
	enc, ok := p.FindValue(data, "Blocks")
	if !ok {
		t.Fatal("expected named block")
	}
	model := pkg.Decode(enc)
	count := 0
	for _, e := range model.Blocks[0].Body {
		if e.Kind == "directive" && e.Key == "IdentityFile" {
			count++
		}
	}
	if count != 3 {
		t.Fatalf("expected 3 IdentityFile entries, got %d", count)
	}
	out, err := p.SetValue(data, "Blocks", enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, data) {
		t.Errorf("multi-IdentityFile round-trip not byte-identical:\ngot:\n%s", out)
	}
}

func TestBlocksNoOpSetIsByteIdentical(t *testing.T) {
	p := blocksParser()
	data := []byte(`# global comment
ServerAliveInterval 60

# owned comment
Host alpha
    HostName alpha.example.com
    User admin

Host beta
    HostName beta.example.com

Host *
    AddKeysToAgent yes
`)
	enc, ok := p.FindValue(data, "Blocks")
	if !ok {
		t.Fatal("expected named blocks")
	}
	out, err := p.SetValue(data, "Blocks", enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, data) {
		t.Errorf("no-op SetValue(Blocks) not byte-identical:\ngot:\n%s\nwant:\n%s", out, data)
	}
}

func TestBlocksRealisticNoOpByteIdentical(t *testing.T) {
	p := blocksParser()
	data := []byte(`# ssh client config
Include ~/.ssh/conf.d/*.conf
ServerAliveInterval 60

# work boxes
Host work
    HostName work.example.com
    User admin
    IdentityFile ~/.ssh/work_ed25519
    IdentityFile ~/.ssh/work_rsa

Host work
    HostName work-2.example.com

Match exec "test -f /tmp/flag"
    ForwardAgent yes
    User scoped

Host *
    AddKeysToAgent yes
    Compression no
`)
	enc, ok := p.FindValue(data, "Blocks")
	if !ok {
		t.Fatal("expected named blocks")
	}
	out, err := p.SetValue(data, "Blocks", enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, data) {
		t.Errorf("realistic no-op SetValue(Blocks) not byte-identical:\ngot:\n%s\nwant:\n%s", out, data)
	}
}

func TestBlocksSingleDirectiveEditIsSurgical(t *testing.T) {
	p := blocksParser()
	data := []byte(`# leading
ServerAliveInterval 60

Host alpha
    # alpha note
    HostName alpha.example.com
    User admin

Host beta
    HostName beta.example.com

Host *
    AddKeysToAgent yes
`)
	enc, ok := p.FindValue(data, "Blocks")
	if !ok {
		t.Fatal("expected named blocks")
	}
	model := pkg.Decode(enc)
	for bi := range model.Blocks {
		if model.Blocks[bi].Header != "alpha" {
			continue
		}
		for ei := range model.Blocks[bi].Body {
			e := &model.Blocks[bi].Body[ei]
			if e.Kind == "directive" && e.Key == "User" {
				e.Values = []string{"root"}
			}
		}
	}
	out, err := p.SetValue(data, "Blocks", pkg.Encode(model))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(string(data), "    User admin", "    User root", 1)
	if string(out) != want {
		t.Errorf("edit was not surgical:\ngot:\n%s\nwant:\n%s", out, want)
	}
}

func TestBlocksIncludeLinesPreserved(t *testing.T) {
	p := blocksParser()
	data := []byte(`Include ~/.ssh/conf.d/*.conf
ServerAliveInterval 60

Host alpha
    HostName alpha.example.com

Include ~/.ssh/extra
Host beta
    HostName beta.example.com

Host *
    Include ~/.ssh/wild.conf
`)
	enc, ok := p.FindValue(data, "Blocks")
	if !ok {
		t.Fatal("expected named blocks")
	}
	out, err := p.SetValue(data, "Blocks", enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, data) {
		t.Errorf("Include lines not preserved verbatim:\ngot:\n%s\nwant:\n%s", out, data)
	}
}

func TestBlocksReorderCarriesCommentsAndKeepsWildLast(t *testing.T) {
	p := blocksParser()
	data := []byte(`Host alpha
    HostName alpha.example.com

# beta is special
Host beta
    HostName beta.example.com

Host *
    AddKeysToAgent yes
`)
	enc, ok := p.FindValue(data, "Blocks")
	if !ok {
		t.Fatal("expected named blocks")
	}
	model := pkg.Decode(enc)
	if len(model.Blocks) != 2 {
		t.Fatalf("expected 2 named blocks, got %d", len(model.Blocks))
	}
	model.Blocks[0], model.Blocks[1] = model.Blocks[1], model.Blocks[0]
	out, err := p.SetValue(data, "Blocks", pkg.Encode(model))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)

	betaAt := strings.Index(text, "Host beta")
	alphaAt := strings.Index(text, "Host alpha")
	wildAt := strings.Index(text, "Host *")
	if betaAt < 0 || alphaAt < 0 || wildAt < 0 {
		t.Fatalf("blocks missing after reorder:\n%s", text)
	}
	if betaAt >= alphaAt {
		t.Errorf("beta should precede alpha after reorder:\n%s", text)
	}
	if alphaAt >= wildAt {
		t.Errorf("Host * must stay last (lowest precedence):\n%s", text)
	}
	// the owned comment travels with beta.
	if idx := strings.Index(text, "# beta is special"); idx < 0 || idx > betaAt {
		t.Errorf("owned comment did not travel with beta block:\n%s", text)
	}
}

func TestBlocksDeleteKeepsFlatStable(t *testing.T) {
	p := blocksParser()
	data := []byte(`ServerAliveInterval 60

Host alpha
    HostName alpha.example.com

Host *
    AddKeysToAgent yes
`)
	out, err := p.DeleteKey(data, "Blocks")
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if strings.Contains(text, "Host alpha") {
		t.Errorf("named block should be deleted:\n%s", text)
	}
	if !strings.Contains(text, "ServerAliveInterval 60") || !strings.Contains(text, "Host *") {
		t.Errorf("flat region / Host * must remain:\n%s", text)
	}
	if _, ok := p.FindValue(out, "Blocks"); ok {
		t.Error("Blocks should be unconfigured after delete")
	}
}

func TestBlocksFindLine(t *testing.T) {
	p := blocksParser()
	data := []byte("ServerAliveInterval 60\n\nHost *\n    AddKeysToAgent yes\n\nHost alpha\n    HostName a\n")
	line, ok := p.FindLine(data, "Blocks")
	if !ok {
		t.Fatal("expected a line for first named block")
	}
	// "Host alpha" is line index 5.
	if line != 5 {
		t.Errorf("FindLine(Blocks) = %d, want 5", line)
	}
}

func TestFlatInsertNoWildCreatesHostStarLast(t *testing.T) {
	p := blocksParser()
	data := []byte("Host alpha\n    HostName alpha.example.com\n")
	out, err := p.SetValue(data, "ForwardAgent", "yes")
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if strings.HasPrefix(text, "ForwardAgent") {
		t.Errorf("flat field must not be prepended at root:\n%s", text)
	}
	alphaAt := strings.Index(text, "Host alpha")
	wildAt := strings.Index(text, "Host *")
	if wildAt < 0 {
		t.Fatalf("a Host * block should have been created:\n%s", text)
	}
	if alphaAt >= wildAt {
		t.Errorf("Host * should be placed after host-specific blocks:\n%s", text)
	}
	if !strings.Contains(text, "    ForwardAgent yes") {
		t.Errorf("directive should live in the new Host * block:\n%s", text)
	}
	got, ok := p.FindValue(out, "ForwardAgent")
	if !ok || got != "yes" {
		t.Errorf("inserted value not found: %q, %v", got, ok)
	}
}
