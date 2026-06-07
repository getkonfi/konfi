package editors

import (
	"strings"
	"testing"

	"github.com/eminert/konfi/pkg"

	tea "charm.land/bubbletea/v2"
)

// sshFixture is a representative Host/Match config used across block tests.
const sshFixture = `Host github.com
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519

# bastion access
Match host bastion exec "test -f /tmp/ok"
    ForwardAgent yes
`

// blockPalette is a small directive palette mirroring an ssh schema.
func blockPalette() []pkg.Field {
	return []pkg.Field{
		{Key: "HostName", Label: "HostName", Type: "string"},
		{Key: "User", Label: "User", Type: "string"},
		{Key: "Port", Label: "Port", Type: "number"},
		{Key: "ForwardAgent", Label: "ForwardAgent", Type: "enum", Options: []string{"yes", "no"}},
		{Key: "IdentityFile", Label: "IdentityFile", Type: "string"},
	}
}

func encodedFixture(t *testing.T) string {
	t.Helper()
	m := pkg.Parse([]byte(sshFixture), []string{"Host", "Match"}, nil)
	return pkg.Encode(m)
}

func newBlockEditor(t *testing.T, encoded string) *blockEditor {
	t.Helper()
	e := &blockEditor{}
	e.Init(pkg.Field{Widget: "blocklist", BlockPalette: blockPalette()}, encoded, testTheme())
	return e
}

// TestBlockEditor_NoEditIdentity proves Decode∘Encode round-trips: with no
// edits, Value() equals the exact encoded input (byte-stable no-op save).
func TestBlockEditor_NoEditIdentity(t *testing.T) {
	enc := encodedFixture(t)
	e := newBlockEditor(t, enc)
	if got := e.Value(); got != enc {
		t.Fatalf("no-edit Value() != input encoding\n got=%q\nwant=%q", got, enc)
	}
}

// TestBlockEditor_EmptyInit starts from an empty value.
func TestBlockEditor_EmptyInit(t *testing.T) {
	e := newBlockEditor(t, "")
	if len(e.model.Blocks) != 0 {
		t.Fatalf("empty init produced %d blocks, want 0", len(e.model.Blocks))
	}
}

// TestBlockEditor_EscCommits proves esc at the block list finishes the editor
// committing (done, not canceled) so edits are remembered, and a no-op esc
// leaves the encoding byte-stable.
func TestBlockEditor_EscCommits(t *testing.T) {
	enc := encodedFixture(t)
	e := newBlockEditor(t, enc)
	_, done, canceled := e.Update(keyMsg("esc"))
	if !done || canceled {
		t.Fatalf("esc at block list: done=%v canceled=%v, want done=true canceled=false", done, canceled)
	}
	if got := e.Value(); got != enc {
		t.Fatalf("no-op esc changed the value:\n got=%q\nwant=%q", got, enc)
	}
}

// TestBlockEditor_LeftCommits proves left arrow behaves like esc at the top.
func TestBlockEditor_LeftCommits(t *testing.T) {
	e := newBlockEditor(t, encodedFixture(t))
	_, done, canceled := e.Update(keyMsg("left"))
	if !done || canceled {
		t.Fatalf("left at block list: done=%v canceled=%v, want done=true canceled=false", done, canceled)
	}
}

// TestBlockEditor_LeftFromBodyReturnsToList proves left exits the body view to
// the block list without finishing the whole editor.
func TestBlockEditor_LeftFromBodyReturnsToList(t *testing.T) {
	e := newBlockEditor(t, encodedFixture(t))
	e.Update(keyMsg("enter")) // open first block body
	if e.mode != modeBody {
		t.Fatalf("expected modeBody after enter, got %v", e.mode)
	}
	_, done, canceled := e.Update(keyMsg("left"))
	if done || canceled {
		t.Fatalf("left in body should not finish editor: done=%v canceled=%v", done, canceled)
	}
	if e.mode != modeBlockList {
		t.Fatalf("expected modeBlockList after left in body, got %v", e.mode)
	}
}

// TestBlockEditor_AddBlock adds a Host block via the add flow.
func TestBlockEditor_AddBlock(t *testing.T) {
	e := newBlockEditor(t, encodedFixture(t))
	start := len(e.model.Blocks)

	e.Update(keyMsg("a"))             // enter add flow (opener choice)
	e.Update(keyMsg("enter"))         // choose Host (idx 0)
	for _, r := range "example.com" { // type header
		e.Update(keyMsg(string(r)))
	}
	e.Update(keyMsg("enter")) // commit

	if len(e.model.Blocks) != start+1 {
		t.Fatalf("add: got %d blocks, want %d", len(e.model.Blocks), start+1)
	}
	last := e.model.Blocks[len(e.model.Blocks)-1]
	if last.Opener != "Host" || last.Header != "example.com" {
		t.Fatalf("new block = %q %q, want Host example.com", last.Opener, last.Header)
	}

	// the new block round-trips through Decode(Encode(...))
	m := pkg.Decode(e.Value())
	if len(m.Blocks) != start+1 {
		t.Fatalf("encoded model has %d blocks, want %d", len(m.Blocks), start+1)
	}
}

// TestBlockEditor_DeleteBlock removes the selected block.
func TestBlockEditor_DeleteBlock(t *testing.T) {
	e := newBlockEditor(t, encodedFixture(t))
	start := len(e.model.Blocks)
	first := e.model.Blocks[0].Header

	e.Update(keyMsg("d"))
	if len(e.model.Blocks) != start-1 {
		t.Fatalf("delete: got %d blocks, want %d", len(e.model.Blocks), start-1)
	}
	if e.model.Blocks[0].Header == first {
		t.Fatalf("delete did not remove first block (header %q still present)", first)
	}
}

// TestBlockEditor_Reorder swaps block order with J/K.
func TestBlockEditor_Reorder(t *testing.T) {
	e := newBlockEditor(t, encodedFixture(t))
	if len(e.model.Blocks) < 2 {
		t.Fatalf("fixture needs >= 2 blocks, got %d", len(e.model.Blocks))
	}
	first, second := e.model.Blocks[0].Header, e.model.Blocks[1].Header

	e.Update(keyMsg("J")) // move first down
	if e.model.Blocks[0].Header != second || e.model.Blocks[1].Header != first {
		t.Fatalf("after J: order = [%q,%q], want [%q,%q]",
			e.model.Blocks[0].Header, e.model.Blocks[1].Header, second, first)
	}
	if e.blockCursor != 1 {
		t.Fatalf("after J: cursor = %d, want 1", e.blockCursor)
	}

	e.Update(keyMsg("K")) // move back up
	if e.model.Blocks[0].Header != first || e.model.Blocks[1].Header != second {
		t.Fatalf("after K: order not restored: [%q,%q]",
			e.model.Blocks[0].Header, e.model.Blocks[1].Header)
	}

	// order change survives the encode channel
	m := pkg.Decode(e.Value())
	if m.Blocks[0].Header != first {
		t.Fatalf("encoded order wrong: first = %q, want %q", m.Blocks[0].Header, first)
	}
}

// directiveValues returns the value tokens for the first directive with key in
// the given block.
func directiveValues(b pkg.Block, key string) ([]string, bool) {
	for _, en := range b.Body {
		if en.Kind == "directive" && strings.EqualFold(en.Key, key) {
			return en.Values, true
		}
	}
	return nil, false
}

// TestBlockEditor_EditDirectiveValue edits a directive via the nested editor.
func TestBlockEditor_EditDirectiveValue(t *testing.T) {
	e := newBlockEditor(t, encodedFixture(t))

	e.Update(keyMsg("enter")) // expand first block (github.com)
	if e.mode != modeBody {
		t.Fatalf("enter did not switch to body view")
	}
	// cursor is on the first directive (HostName). edit it.
	e.Update(keyMsg("enter")) // open nested editor
	if !e.editing || e.nested == nil {
		t.Fatalf("enter did not open nested editor")
	}
	// clear current value and type a new one.
	for range "github.com" {
		e.Update(keyMsg("backspace"))
	}
	for _, r := range "gh.example" {
		e.Update(keyMsg(string(r)))
	}
	e.Update(keyMsg("enter")) // commit nested

	if e.editing {
		t.Fatalf("nested editor still active after commit")
	}
	vals, ok := directiveValues(e.model.Blocks[0], "HostName")
	if !ok {
		t.Fatalf("HostName directive missing")
	}
	if strings.Join(vals, " ") != "gh.example" {
		t.Fatalf("HostName = %q, want gh.example", strings.Join(vals, " "))
	}
	// other directives untouched
	if v, _ := directiveValues(e.model.Blocks[0], "User"); strings.Join(v, " ") != "git" {
		t.Fatalf("User changed unexpectedly: %q", v)
	}
}

// TestBlockEditor_NestedCancelDiscards verifies esc in the nested editor
// discards the edit.
func TestBlockEditor_NestedCancelDiscards(t *testing.T) {
	e := newBlockEditor(t, encodedFixture(t))
	e.Update(keyMsg("enter")) // expand block
	e.Update(keyMsg("enter")) // open nested editor on HostName

	for range "github.com" {
		e.Update(keyMsg("backspace"))
	}
	for _, r := range "discarded" {
		e.Update(keyMsg(string(r)))
	}
	e.Update(keyMsg("esc")) // cancel nested

	if e.editing {
		t.Fatalf("nested editor still active after esc")
	}
	vals, _ := directiveValues(e.model.Blocks[0], "HostName")
	if strings.Join(vals, " ") != "github.com" {
		t.Fatalf("HostName changed despite cancel: %q", vals)
	}
}

// TestBlockEditor_NestedHeightChanges verifies Height() grows while the nested
// editor is active.
func TestBlockEditor_NestedHeightChanges(t *testing.T) {
	e := newBlockEditor(t, encodedFixture(t))
	e.Update(keyMsg("enter")) // expand block (body view)
	base := e.Height()

	// move to the enum directive? simplest: edit HostName (string, height 0) is
	// not useful; instead edit ForwardAgent which is enum with positive height.
	// navigate to a directive whose nested editor has positive height: User is
	// string (height 0). use the enum block instead.
	e.Update(keyMsg("esc")) // back to list

	// expand the Match block (has ForwardAgent enum)
	e.Update(keyMsg("j"))     // cursor -> second block
	e.Update(keyMsg("enter")) // expand
	beforeEdit := e.Height()
	e.Update(keyMsg("enter")) // open nested enum editor
	if !e.editing {
		t.Fatalf("nested editor not active")
	}
	afterEdit := e.Height()
	if afterEdit <= beforeEdit {
		t.Fatalf("height did not grow while editing: before=%d after=%d", beforeEdit, afterEdit)
	}
	_ = base
}

// TestBlockEditor_AddDirective adds a directive via the palette menu.
func TestBlockEditor_AddDirective(t *testing.T) {
	e := newBlockEditor(t, encodedFixture(t))
	e.Update(keyMsg("enter")) // expand first block

	before := countDirectives(e.model.Blocks[0])
	e.Update(keyMsg("a")) // open palette menu
	if !e.addingDirective {
		t.Fatalf("a did not open directive palette")
	}
	// select Port (index 2)
	e.Update(keyMsg("j"))
	e.Update(keyMsg("j"))
	e.Update(keyMsg("enter")) // add Port; opens nested editor
	for _, r := range "2222" {
		e.Update(keyMsg(string(r)))
	}
	e.Update(keyMsg("enter")) // commit value

	after := countDirectives(e.model.Blocks[0])
	if after != before+1 {
		t.Fatalf("add directive: %d -> %d, want +1", before, after)
	}
	vals, ok := directiveValues(e.model.Blocks[0], "Port")
	if !ok || strings.Join(vals, " ") != "2222" {
		t.Fatalf("Port directive = %v ok=%v, want 2222", vals, ok)
	}
}

// TestBlockEditor_DeleteDirective removes a directive from a block.
func TestBlockEditor_DeleteDirective(t *testing.T) {
	e := newBlockEditor(t, encodedFixture(t))
	e.Update(keyMsg("enter")) // expand first block

	before := countDirectives(e.model.Blocks[0])
	e.Update(keyMsg("d")) // delete directive under cursor (HostName)
	after := countDirectives(e.model.Blocks[0])
	if after != before-1 {
		t.Fatalf("delete directive: %d -> %d, want -1", before, after)
	}
	if _, ok := directiveValues(e.model.Blocks[0], "HostName"); ok {
		t.Fatalf("HostName still present after delete")
	}
	// remaining directives intact
	if v, _ := directiveValues(e.model.Blocks[0], "User"); strings.Join(v, " ") != "git" {
		t.Fatalf("User changed after delete: %q", v)
	}
}

func countDirectives(b pkg.Block) int {
	n := 0
	for _, en := range b.Body {
		if en.Kind == "directive" {
			n++
		}
	}
	return n
}

// TestBlockEditor_HostHeaderMultiPattern edits a Host header as a space-separated
// pattern set.
func TestBlockEditor_HostHeaderMultiPattern(t *testing.T) {
	e := newBlockEditor(t, encodedFixture(t))
	// first block is Host github.com. edit its header.
	e.Update(keyMsg("h"))
	if !e.editingHeader {
		t.Fatalf("h did not start header edit")
	}
	// replace value
	for range "github.com" {
		e.Update(keyMsg("backspace"))
	}
	for _, r := range "gh1 gh2 !gh3" {
		e.Update(keyMsg(string(r)))
	}
	e.Update(keyMsg("enter"))

	if e.editingHeader {
		t.Fatalf("header edit still active after enter")
	}
	if got := e.model.Blocks[0].Header; got != "gh1 gh2 !gh3" {
		t.Fatalf("Host header = %q, want %q", got, "gh1 gh2 !gh3")
	}
	// header survives the encode channel and the opener line is rebuilt
	m := pkg.Decode(e.Value())
	if m.Blocks[0].Header != "gh1 gh2 !gh3" {
		t.Fatalf("encoded Host header = %q", m.Blocks[0].Header)
	}
}

// TestBlockEditor_MatchHeaderRawPreservesExec verifies a Match header is stored
// verbatim, preserving exec "..." quoting (never field-split/normalized).
func TestBlockEditor_MatchHeaderRawPreservesExec(t *testing.T) {
	e := newBlockEditor(t, encodedFixture(t))
	// second block is the Match block.
	e.Update(keyMsg("j")) // cursor -> Match block
	e.Update(keyMsg("h")) // edit header
	if !e.editingHeader || e.headerForHost {
		t.Fatalf("Match header edit not in raw (non-host) mode")
	}
	// retype a header containing an exec with a quoted command and extra spaces.
	cur := e.model.Blocks[1].Header
	for range cur {
		e.Update(keyMsg("backspace"))
	}
	const want = `host bastion exec "test -f /tmp/ok"`
	for _, r := range want {
		e.Update(keyMsg(string(r)))
	}
	e.Update(keyMsg("enter"))

	if got := e.model.Blocks[1].Header; got != want {
		t.Fatalf("Match header = %q, want %q (must be verbatim)", got, want)
	}
	m := pkg.Decode(e.Value())
	if m.Blocks[1].Header != want {
		t.Fatalf("encoded Match header = %q, want %q", m.Blocks[1].Header, want)
	}
}

// TestBlockEditor_EnterFocusesNested ensures entering edit focuses the nested
// editor (it consumes typed characters rather than the block editor).
func TestBlockEditor_EnterFocusesNested(t *testing.T) {
	e := newBlockEditor(t, encodedFixture(t))
	e.Update(keyMsg("enter")) // expand block
	e.Update(keyMsg("enter")) // open nested editor on HostName

	// typing "j" should go to the nested editor (append a char), not navigate.
	e.Update(keyMsg("j"))
	if se, ok := e.nested.(*stringEditor); ok {
		if !strings.HasSuffix(se.input.Value(), "j") {
			t.Fatalf("nested input did not receive 'j': %q", se.input.Value())
		}
	} else {
		t.Fatalf("nested editor is %T, want *stringEditor", e.nested)
	}
}

// TestBlockEditor_RestrictListToString verifies a list/multi palette field is
// edited as a single string occurrence (no nested list editor).
func TestBlockEditor_RestrictListToString(t *testing.T) {
	f := pkg.Field{Key: "IdentityFile", Type: "list"}
	got := restrictToSingleValue(f)
	if got.Type != "string" || got.Widget != "" {
		t.Fatalf("list field restricted to %+v, want string/no-widget", got)
	}
}

// guard: blockEditor implements the optional editor interfaces it claims.
var (
	_ FieldEditor  = (*blockEditor)(nil)
	_ OffsetEditor = (*blockEditor)(nil)
	_ Interactor   = (*blockEditor)(nil)
)

// guard: blockEditor must NOT be a MultiValueEditor (value is one opaque string).
func TestBlockEditor_NotMultiValue(t *testing.T) {
	var fe FieldEditor = (*blockEditor)(nil)
	if _, ok := fe.(MultiValueEditor); ok {
		t.Fatalf("blockEditor must not implement MultiValueEditor")
	}
}

// guard: typing into the nested editor uses the textinput, ensuring our keyMsg
// helper drives real input (sanity for the other tests).
func TestBlockEditor_TextInputDriver(t *testing.T) {
	in := newFieldInput(testTheme())
	cmd := in.Focus()
	_ = cmd
	var msg tea.Msg = keyMsg("x")
	in, _ = in.Update(msg)
	if in.Value() != "x" {
		t.Fatalf("textinput did not consume keyMsg: %q", in.Value())
	}
}
