package ui

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/eminert/konfi/konfables"
	"github.com/eminert/konfi/pkg"
	cfgparse "github.com/eminert/konfi/pkg/parser"
	"github.com/eminert/konfi/setup"
	"github.com/eminert/konfi/ui/editors"

	tea "charm.land/bubbletea/v2"
)

type switchTestKonfable struct {
	name   string
	parser konfables.Parser
	data   []byte
}

func (k *switchTestKonfable) Info() konfables.AppInfo {
	return konfables.AppInfo{Name: k.name, Format: "ghostty"}
}

func (k *switchTestKonfable) Parser() konfables.Parser { return k.parser }
func (k *switchTestKonfable) Schema() ([]byte, error)  { return nil, nil }
func (k *switchTestKonfable) Name() string             { return k.name }
func (k *switchTestKonfable) ConfigPath() string       { return "/tmp/" + k.name + ".conf" }

func (k *switchTestKonfable) Load(context.Context) ([]byte, error) {
	return k.data, nil
}

func (k *switchTestKonfable) Save(_ context.Context, _, data []byte) error {
	k.data = data
	return nil
}

func newContentFocusTestModel(t *testing.T) content {
	t.Helper()

	th := testTheme()
	p := &cfgparse.FlatParser{Split: cfgparse.SplitEquals, Format: cfgparse.FormatEquals}
	k := detailTestKonfable{parser: p, info: konfables.AppInfo{Format: "ghostty"}}
	var data strings.Builder
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&data, "line%d = %d\n", i, i)
	}
	cf, err := pkg.NewConfigFile(context.Background(), &detailTestPersister{data: []byte(data.String())})
	if err != nil {
		t.Fatal(err)
	}

	c := newContent(th)
	c.focused = true
	c.width = 120
	c.height = 8
	c.konfable = k
	c.config = cf
	c.schema = &pkg.Schema{
		Sections: []pkg.Section{{
			Name: "general",
			Fields: []pkg.Field{
				{Key: "line1", Label: "Line 1", Type: "string", Default: "1"},
				{Key: "line12", Label: "Line 12", Type: "string", Default: "12"},
			},
		}},
	}
	c.buildFieldList()
	c.refreshValues()
	c.snapshotOrigValues()
	c.cursor = 2
	c.syncDetail()
	return c
}

func TestContentRightFocusesConfigPaneAndScrollsWithoutMovingFieldCursor(t *testing.T) {
	c := newContentFocusTestModel(t)
	startCursor := c.cursor

	var cmd tea.Cmd
	c, cmd = c.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if cmd != nil {
		t.Fatal("right arrow should not return a command")
	}
	if !c.detailFocused {
		t.Fatal("right arrow on a field did not focus the config pane")
	}
	if c.cursor != startCursor {
		t.Fatalf("right arrow moved field cursor: got %d want %d", c.cursor, startCursor)
	}

	startScroll := c.detail.scrollY
	c, _ = c.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if c.cursor != startCursor {
		t.Fatalf("down in config pane moved field cursor: got %d want %d", c.cursor, startCursor)
	}
	if c.detail.scrollY <= startScroll {
		t.Fatalf("down in config pane did not advance file scroll: got %d want > %d", c.detail.scrollY, startScroll)
	}

	c, _ = c.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if c.detailFocused {
		t.Fatal("left arrow did not return focus to the field list")
	}
}

func TestContentDeleteKeyDeletesConfiguredField(t *testing.T) {
	c := newContentFocusTestModel(t)

	c, _ = c.Update(tea.KeyPressMsg{Code: tea.KeyDelete})

	if _, ok := c.values["line12"]; ok {
		t.Fatal("delete key did not remove field from values")
	}
	if strings.Contains(string(c.config.Content()), "line12") {
		t.Fatalf("delete key did not remove field from config:\n%s", c.config.Content())
	}
}

func TestContentDeleteChangedFieldRevertsBeforeDeleting(t *testing.T) {
	c := newContentFocusTestModel(t)
	c.fields[1].Default = "0"

	commitTestField(t, &c, "line12", "99")

	c, _ = c.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if got := c.values["line12"]; got != "12" {
		t.Fatalf("first backspace value = %q, want loaded value 12", got)
	}
	if strings.Contains(string(c.config.Content()), "line12 = 99") {
		t.Fatalf("first backspace kept edited value in config:\n%s", c.config.Content())
	}
	if c.config.Dirty() {
		t.Fatal("first backspace should restore the loaded file value and clear dirty state")
	}

	c, _ = c.Update(tea.KeyPressMsg{Code: tea.KeyDelete})
	if _, ok := c.values["line12"]; ok {
		t.Fatal("second delete should remove the configured value")
	}
	if strings.Contains(string(c.config.Content()), "line12") {
		t.Fatalf("second delete did not remove field from config:\n%s", c.config.Content())
	}
	if got := stripANSI(c.renderFieldValue(c.fields[1], c.fields[1].Default, true)); got != "0" {
		t.Fatalf("default render after delete = %q, want 0", got)
	}
}

func TestRootDotTogglesConfiguredOnly(t *testing.T) {
	c := newContentFocusTestModel(t)
	r := &root{content: c, focus: paneContent}

	_, _ = r.Update(tea.KeyPressMsg{Text: "."})

	if !r.content.configuredOnly {
		t.Fatal("dot did not toggle configured-only filter")
	}
}

func TestRootTabTogglesChangedOnly(t *testing.T) {
	c := newContentFocusTestModel(t)
	c.applyFieldByKey("line12", "99")
	r := &root{content: c, focus: paneContent}

	_, _ = r.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	if !r.content.changedOnly {
		t.Fatal("tab did not toggle changed-only filter")
	}
	var visibleKeys []string
	for _, row := range r.content.visible {
		if row.isSection {
			continue
		}
		visibleKeys = append(visibleKeys, r.content.fields[row.fieldIdx].Key)
	}
	if !reflect.DeepEqual(visibleKeys, []string{"line12"}) {
		t.Fatalf("visible fields after changed-only = %#v, want [line12]", visibleKeys)
	}
}

func TestRootTabWithNoChangesShowsFeedback(t *testing.T) {
	c := newContentFocusTestModel(t)
	r := &root{content: c, focus: paneContent}

	_, _ = r.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	if !r.content.changedOnly {
		t.Fatal("tab did not toggle changed-only filter")
	}
	if r.status.status != "no unsaved changes" {
		t.Fatalf("status = %q, want no unsaved changes", r.status.status)
	}
	if got := stripANSI(r.content.renderBody(60)); !strings.Contains(got, "no unsaved changes") {
		t.Fatalf("changed-only empty state missing feedback:\n%s", got)
	}
}

func TestSaveResultKeepsSavedStatusThenClears(t *testing.T) {
	c := newContentFocusTestModel(t)
	c.applyFieldByKey("line12", "99")
	if err := c.config.Save(context.Background()); err != nil {
		t.Fatal(err)
	}
	c.changedOnly = true
	c.refilter()
	r := &root{content: c, focus: paneContent, dirtyConfigs: make(map[string]dirtyConfigState)}

	_, _ = r.Update(saveResultMsg{})

	if r.status.status != "saved" {
		t.Fatalf("status = %q, want saved", r.status.status)
	}
	if changes := r.content.pendingChanges(); len(changes) != 0 {
		t.Fatalf("pending changes after save = %#v, want none", changes)
	}
	if len(r.content.visible) != 0 {
		t.Fatalf("changed-only visible rows after save = %#v, want none", r.content.visible)
	}

	r.status.status = "opened docs"
	_, _ = r.Update(statusClearMsg{status: "saved"})
	if r.status.status != "opened docs" {
		t.Fatalf("guarded clear removed newer status: %q", r.status.status)
	}
	_, _ = r.Update(statusClearMsg{status: "opened docs"})
	if r.status.status != "" {
		t.Fatalf("status after matching clear = %q, want empty", r.status.status)
	}
}

func TestSwitchingAppsKeepsUnsavedConfigInSession(t *testing.T) {
	c := newContentFocusTestModel(t)
	c.schemaCache = map[string]*pkg.Schema{"test": c.schema}
	c.applyFieldByKey("line12", "99")
	if c.config == nil || !c.config.Dirty() {
		t.Fatal("test setup did not create a dirty config")
	}

	p := &cfgparse.FlatParser{Split: cfgparse.SplitEquals, Format: cfgparse.FormatEquals}
	other := &switchTestKonfable{name: "other", parser: p, data: []byte("other = 1\n")}
	otherConfig, err := pkg.NewConfigFile(context.Background(), other)
	if err != nil {
		t.Fatal(err)
	}
	r := &root{
		app:          &setup.App{Config: &setup.KonfConfig{}},
		content:      c,
		focus:        paneContent,
		allKonfables: []konfables.Konfable{c.konfable, other},
		dirtyConfigs: make(map[string]dirtyConfigState),
	}

	_, _ = r.Update(AppSelectedMsg{Index: 1, Confirmed: true})
	if _, ok := r.dirtyConfigs["test"]; !ok {
		t.Fatal("dirty config was not stashed before switching apps")
	}
	_, _ = r.Update(appLoadedMsg{appName: "other", config: otherConfig, path: other.ConfigPath()})

	_, _ = r.Update(AppSelectedMsg{Index: 0, Confirmed: true})
	_, _ = r.Update(appLoadedMsg{appName: "test", path: "/tmp/test.conf"})

	if r.content.config == nil || !r.content.config.Dirty() {
		t.Fatal("restored config is not dirty")
	}
	if got := r.content.values["line12"]; got != "99" {
		t.Fatalf("restored line12 = %q, want 99", got)
	}
	changes := r.content.pendingChanges()
	if len(changes) != 1 || changes[0].Key != "line12" || changes[0].OldVal != "12" || changes[0].NewVal != "99" {
		t.Fatalf("restored pending changes = %#v, want line12 12 -> 99", changes)
	}
}

func TestEditorReloadKeepsInTUIEditsForUnchangedFields(t *testing.T) {
	c, persister := newEditorReloadTestContent(t)
	commitTestField(t, &c, "line12", "99")

	persister.data = []byte("line1 = external\nline12 = 12\n")
	r := &root{content: c, focus: paneContent, dirtyConfigs: make(map[string]dirtyConfigState)}

	runEditorReload(t, r)

	if got := r.content.values["line1"]; got != "external" {
		t.Fatalf("line1 = %q, want external", got)
	}
	if got := r.content.values["line12"]; got != "99" {
		t.Fatalf("line12 = %q, want 99", got)
	}
	if !r.content.config.Dirty() {
		t.Fatal("config should remain dirty after replaying in-TUI edit")
	}
	changes := r.content.pendingChanges()
	if len(changes) != 1 || changes[0].Key != "line12" || changes[0].OldVal != "12" || changes[0].NewVal != "99" {
		t.Fatalf("pending changes = %#v, want line12 12 -> 99", changes)
	}
	content := string(r.content.config.Content())
	if !strings.Contains(content, "line1 = external") || !strings.Contains(content, "line12 = 99") {
		t.Fatalf("merged content mismatch:\n%s", content)
	}
}

func TestEditorReloadSkipsInTUIEditWhenEditorChangedSameField(t *testing.T) {
	c, persister := newEditorReloadTestContent(t)
	commitTestField(t, &c, "line1", "10")
	commitTestField(t, &c, "line12", "99")

	persister.data = []byte("line1 = 1\nline12 = 77\n")
	r := &root{content: c, focus: paneContent, dirtyConfigs: make(map[string]dirtyConfigState)}

	runEditorReload(t, r)

	if got := r.content.values["line1"]; got != "10" {
		t.Fatalf("line1 = %q, want 10", got)
	}
	if got := r.content.values["line12"]; got != "77" {
		t.Fatalf("line12 = %q, want editor value 77", got)
	}
	if !r.content.config.Dirty() {
		t.Fatal("config should stay dirty because line1 was replayed")
	}
	changes := r.content.pendingChanges()
	if len(changes) != 1 || changes[0].Key != "line1" || changes[0].OldVal != "1" || changes[0].NewVal != "10" {
		t.Fatalf("pending changes = %#v, want only line1 1 -> 10", changes)
	}
	if r.content.undoStack.Len() != 1 {
		t.Fatalf("undo stack len = %d, want only replayed field history", r.content.undoStack.Len())
	}
	op, ok := r.content.undoStack.Undo()
	if !ok || op.FieldKey != "line1" {
		t.Fatalf("undo op = %+v, ok=%v; want line1 only", op, ok)
	}
}

func TestContentBottomHelpersMatchRequestedKeys(t *testing.T) {
	c := newContentFocusTestModel(t)
	r := &root{content: c, focus: paneContent}

	r.updateHints()

	want := []keyHint{
		{"↑↓", "nav"},
		{"⏎", "edit"},
		{"⌫", "del"},
		{"/", "search"},
		{"c", "copy"},
		{".", "configured"},
		{"⇥", "changed"},
		{"q", "quit"},
		{"esc", "cancel"},
	}
	if !reflect.DeepEqual(r.status.hints, want) {
		t.Fatalf("status hints mismatch\ngot:  %#v\nwant: %#v", r.status.hints, want)
	}
}

func TestContentBottomHelpersMatchMultiSelectKeys(t *testing.T) {
	c := newContentFocusTestModel(t)
	field := pkg.Field{Type: "multi", Options: []string{"bold", "italic"}}
	c.detail.editor = editors.ForField(field)
	c.detail.editor.Init(field, "", testTheme())
	r := &root{content: c, focus: paneContent}

	r.updateHints()

	want := []keyHint{
		{"↑↓", "nav"},
		{"␣", "select"},
		{"⏎", "accept"},
		{"esc", "cancel"},
	}
	if !reflect.DeepEqual(r.status.hints, want) {
		t.Fatalf("status hints mismatch\ngot:  %#v\nwant: %#v", r.status.hints, want)
	}
}

func TestSidebarRightSelectsCurrentApp(t *testing.T) {
	s := newSidebar([]sidebarItem{
		{name: "home", installed: true, home: true},
		{name: "ghostty", installed: true},
	}, testTheme())
	s.focused = true
	s.cursor = 1
	r := &root{sidebar: s, focus: paneSidebar}

	_, cmd := r.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if cmd == nil {
		t.Fatal("right arrow from sidebar did not emit app selection")
	}
	msg := cmd()
	selected, ok := msg.(AppSelectedMsg)
	if !ok {
		t.Fatalf("right arrow emitted %T, want AppSelectedMsg", msg)
	}
	if selected.Index != 0 || !selected.Confirmed {
		t.Fatalf("selection = {Index:%d Confirmed:%v}, want {Index:0 Confirmed:true}", selected.Index, selected.Confirmed)
	}
}

func TestNumberKeysDoNotJumpApps(t *testing.T) {
	s := newSidebar([]sidebarItem{
		{name: "home", installed: true, home: true},
		{name: "ghostty", installed: true},
	}, testTheme())
	s.focused = true
	r := &root{
		sidebar:      s,
		content:      newContent(testTheme()),
		focus:        paneSidebar,
		allKonfables: []konfables.Konfable{detailTestKonfable{}},
		dirtyConfigs: make(map[string]dirtyConfigState),
	}

	_, cmd := r.Update(tea.KeyPressMsg{Text: "1"})
	if cmd != nil {
		msg := cmd()
		t.Fatalf("number key emitted %T, want no app jump", msg)
	}
}

func TestReviewShortcutRemovedForDirtyConfig(t *testing.T) {
	c := newContentFocusTestModel(t)
	c.applyFieldByKey("line12", "99")
	r := &root{
		content:      c,
		focus:        paneContent,
		allKonfables: []konfables.Konfable{c.konfable},
		dirtyConfigs: make(map[string]dirtyConfigState),
	}

	_, _ = r.Update(tea.KeyPressMsg{Text: "r"})

	if r.content.changedOnly {
		t.Fatal("r toggled changed-only filter; tab should own that flow")
	}
	if r.status.status != "" {
		t.Fatalf("status = %q, want empty", r.status.status)
	}
}

func newEditorReloadTestContent(t *testing.T) (content, *detailTestPersister) {
	t.Helper()

	th := testTheme()
	p := &cfgparse.FlatParser{Split: cfgparse.SplitEquals, Format: cfgparse.FormatEquals}
	k := detailTestKonfable{parser: p, info: konfables.AppInfo{Format: "ghostty"}}
	persister := &detailTestPersister{data: []byte("line1 = 1\nline12 = 12\n")}
	cf, err := pkg.NewConfigFile(context.Background(), persister)
	if err != nil {
		t.Fatal(err)
	}

	c := newContent(th)
	c.focused = true
	c.width = 120
	c.height = 8
	c.konfable = k
	c.config = cf
	c.schema = &pkg.Schema{
		Sections: []pkg.Section{{
			Name: "general",
			Fields: []pkg.Field{
				{Key: "line1", Label: "Line 1", Type: "string", Default: "1"},
				{Key: "line12", Label: "Line 12", Type: "string", Default: "12"},
			},
		}},
	}
	c.buildFieldList()
	c.refreshValues()
	c.snapshotOrigValues()
	c.syncDetail()
	return c, persister
}

func commitTestField(t *testing.T, c *content, key, value string) {
	t.Helper()

	for i := range c.fields {
		if c.fields[i].Key == key {
			c.detail.editField = i
			c.detail.editOrigVal = c.values[key]
			c.commitEdit(value)
			return
		}
	}
	t.Fatalf("field %q not found", key)
}

func runEditorReload(t *testing.T, r *root) {
	t.Helper()

	_, cmd := r.Update(EditorExitMsg{})
	if cmd == nil {
		t.Fatal("EditorExitMsg did not return reload command")
	}
	msg := cmd()
	if _, ok := msg.(reloadResultMsg); !ok {
		t.Fatalf("reload command returned %T, want reloadResultMsg", msg)
	}
	_, _ = r.Update(msg)
}
