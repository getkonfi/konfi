package editors

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/getkonfi/konfi/pkg"
	"github.com/getkonfi/konfi/theme"

	tea "charm.land/bubbletea/v2"
)

func testTheme() *theme.Theme {
	p := theme.PaletteByName("catppuccin")
	return theme.NewTheme(p)
}

func keyMsg(key string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: 0, Text: key}
}

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

// ── toggleMapEditor ─────────────────────────────────────────────────────────

func TestToggleMap_EmptyString(t *testing.T) {
	e := &toggleMapEditor{}
	e.Init(pkg.Field{}, "", testTheme())
	v := e.Value()
	// should produce valid JSON
	var m map[string]bool
	if err := json.Unmarshal([]byte(v), &m); err != nil {
		t.Fatalf("empty init produced invalid JSON: %q err=%v", v, err)
	}
}

func TestToggleMap_InvalidJSON(t *testing.T) {
	th := testTheme()
	cases := []struct {
		name  string
		input string
	}{
		{"malformed brace", `{invalid`},
		{"null literal", `null`},
		{"json array", `[1,2,3]`},
		{"json string", `"hello"`},
		{"json number", `42`},
		{"nested object", `{"a":{"b":true}}`},
		{"bool values non-bool", `{"a":"yes","b":1}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := &toggleMapEditor{}
			e.Init(pkg.Field{}, tc.input, th)
			v := e.Value()
			var m map[string]bool
			if err := json.Unmarshal([]byte(v), &m); err != nil {
				t.Fatalf("input %q → invalid JSON output: %q err=%v", tc.input, v, err)
			}
			// should not panic on View
			_ = e.View(80)
			_ = e.Height()
		})
	}
}

func TestToggleMap_DuplicateKeysInJSON(t *testing.T) {
	// JSON with duplicate keys — Go's json.Unmarshal takes the last value
	e := &toggleMapEditor{}
	e.Init(pkg.Field{}, `{"dup":true,"dup":false}`, testTheme())
	v := e.Value()
	var m map[string]bool
	if err := json.Unmarshal([]byte(v), &m); err != nil {
		t.Fatalf("duplicate keys produced invalid JSON: %q", v)
	}
	// should have exactly one "dup" key
	count := strings.Count(v, `"dup"`)
	if count != 1 {
		t.Fatalf("expected 1 occurrence of dup key, got %d in %q", count, v)
	}
}

func TestToggleMap_SpecialCharKeys(t *testing.T) {
	th := testTheme()
	// keys with quotes, backslashes, newlines
	input := `{"key\"with\\quotes":true,"line\nbreak":false,"normal":true}`
	e := &toggleMapEditor{}
	e.Init(pkg.Field{}, input, th)
	v := e.Value()
	var m map[string]bool
	if err := json.Unmarshal([]byte(v), &m); err != nil {
		t.Fatalf("special char keys produced invalid JSON: %q err=%v", v, err)
	}
	_ = e.View(80)
}

func TestToggleMap_LargeMap(t *testing.T) {
	m := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		m["key_"+strings.Repeat("x", 100)+"_"+string(rune('a'+i%26))] = i%2 == 0
	}
	data, _ := json.Marshal(m)
	e := &toggleMapEditor{}
	e.Init(pkg.Field{}, string(data), testTheme())
	v := e.Value()
	var out map[string]bool
	if err := json.Unmarshal([]byte(v), &out); err != nil {
		t.Fatalf("large map produced invalid JSON: err=%v", err)
	}
	_ = e.View(80)
	_ = e.Height()
}

func TestToggleMap_DoubleInit(t *testing.T) {
	th := testTheme()
	e := &toggleMapEditor{}
	e.Init(pkg.Field{}, `{"a":true}`, th)
	e.Init(pkg.Field{}, `{"b":false}`, th)
	v := e.Value()
	var m map[string]bool
	if err := json.Unmarshal([]byte(v), &m); err != nil {
		t.Fatalf("double init produced invalid JSON: %q", v)
	}
	// after second init, should only have entries from second init (no leftover from first)
	if _, ok := m["a"]; ok {
		t.Fatalf("double init leaked entries from first init: %q", v)
	}
}

func TestToggleMap_EmptyJSON(t *testing.T) {
	e := &toggleMapEditor{}
	e.Init(pkg.Field{}, `{}`, testTheme())
	if len(e.entries) != 0 {
		t.Fatalf("expected 0 entries from {}, got %d", len(e.entries))
	}
	_ = e.View(80)
}

func TestToggleMap_DeleteOnEmpty(t *testing.T) {
	e := &toggleMapEditor{}
	e.Init(pkg.Field{}, `{}`, testTheme())
	// press 'd' on empty list — should not panic
	e.Update(tea.KeyPressMsg{Code: 0, Text: "d"})
	_ = e.View(80)
}

func TestToggleMap_ToggleOnEmpty(t *testing.T) {
	e := &toggleMapEditor{}
	e.Init(pkg.Field{}, `{}`, testTheme())
	// press space on empty list — should not panic
	e.Update(tea.KeyPressMsg{Code: 0, Text: " "})
	_ = e.View(80)
}

// ── structListEditor ────────────────────────────────────────────────────────

func TestStructList_EmptySchema(t *testing.T) {
	e := &structListEditor{}
	field := pkg.Field{ItemSchema: nil, Separator: "="}
	// empty item_schema — Init should not panic
	e.Init(field, "", testTheme())
	_ = e.View(80)
	_ = e.Value()
	_ = e.Height()
}

func TestStructList_EmptySeparator(t *testing.T) {
	e := &structListEditor{}
	field := pkg.Field{
		ItemSchema: []pkg.FieldPart{{Name: "key", Type: "string"}, {Name: "val", Type: "string"}},
		Separator:  "",
	}
	e.Init(field, "hello=world", testTheme())
	// should default separator to "="
	if e.sep != "=" {
		t.Fatalf("expected default separator '=', got %q", e.sep)
	}
	v := e.Value()
	if v == "" {
		t.Fatalf("expected non-empty value, got empty")
	}
}

func TestStructList_SeparatorNotInValue(t *testing.T) {
	e := &structListEditor{}
	field := pkg.Field{
		ItemSchema: []pkg.FieldPart{{Name: "key", Type: "string"}, {Name: "val", Type: "string"}},
		Separator:  "=",
	}
	// value with no separator — parseLine should not panic, second part should be ""
	e.Init(field, "noseparator", testTheme())
	if len(e.items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(e.items))
	}
	if len(e.items[0]) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(e.items[0]))
	}
	if e.items[0][1] != "" {
		t.Fatalf("expected empty second part, got %q", e.items[0][1])
	}
}

func TestStructList_WrongPartCount(t *testing.T) {
	e := &structListEditor{}
	field := pkg.Field{
		ItemSchema: []pkg.FieldPart{
			{Name: "a", Type: "string"},
			{Name: "b", Type: "string"},
			{Name: "c", Type: "string"},
		},
		Separator: ":",
	}
	// only two parts instead of three
	e.Init(field, "one:two", testTheme())
	if len(e.items[0]) != 3 {
		t.Fatalf("expected 3 parts (padded), got %d", len(e.items[0]))
	}
}

func TestStructList_ExtraParts(t *testing.T) {
	e := &structListEditor{}
	field := pkg.Field{
		ItemSchema: []pkg.FieldPart{
			{Name: "a", Type: "string"},
			{Name: "b", Type: "string"},
		},
		Separator: ":",
	}
	// three parts but schema only has two — SplitN should group last parts together
	e.Init(field, "one:two:three", testTheme())
	if len(e.items[0]) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(e.items[0]))
	}
	// second part should be "two:three"
	if e.items[0][1] != "two:three" {
		t.Fatalf("expected second part to contain remainder, got %q", e.items[0][1])
	}
}

func TestStructList_EmptyValue(t *testing.T) {
	e := &structListEditor{}
	field := pkg.Field{
		ItemSchema: []pkg.FieldPart{{Name: "key", Type: "string"}},
		Separator:  "=",
	}
	e.Init(field, "", testTheme())
	if len(e.items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(e.items))
	}
	v := e.Value()
	if v != "" {
		t.Fatalf("expected empty value, got %q", v)
	}
}

func TestStructList_MultilineValues(t *testing.T) {
	e := &structListEditor{}
	field := pkg.Field{
		ItemSchema: []pkg.FieldPart{{Name: "key", Type: "string"}, {Name: "val", Type: "string"}},
		Separator:  "=",
	}
	e.Init(field, "a=1\nb=2\n\nc=3", testTheme())
	// blank lines should be skipped
	if len(e.items) != 3 {
		t.Fatalf("expected 3 items (blank lines skipped), got %d", len(e.items))
	}
}

func TestStructList_DoubleInit(t *testing.T) {
	th := testTheme()
	e := &structListEditor{}
	field := pkg.Field{
		ItemSchema: []pkg.FieldPart{{Name: "k", Type: "string"}},
		Separator:  "=",
	}
	e.Init(field, "first", th)
	e.Init(field, "second", th)
	if len(e.items) != 1 || e.items[0][0] != "second" {
		t.Fatalf("double init did not reset: items=%v", e.items)
	}
}

func TestStructList_DeleteOnEmpty(t *testing.T) {
	e := &structListEditor{}
	field := pkg.Field{
		ItemSchema: []pkg.FieldPart{{Name: "k", Type: "string"}},
		Separator:  "=",
	}
	e.Init(field, "", testTheme())
	e.Update(tea.KeyPressMsg{Code: 0, Text: "d"})
	_ = e.View(80)
}

func TestStructList_ZeroLenItemSchema_StartEdit(t *testing.T) {
	// zero-length item_schema and pressing 'a' to add — should not panic
	e := &structListEditor{}
	field := pkg.Field{ItemSchema: nil, Separator: "="}
	e.Init(field, "", testTheme())

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on startEdit with nil schema: %v", r)
		}
	}()

	// pressing 'a' triggers startEdit → setupStep which indexes e.schema[0]
	e.Update(tea.KeyPressMsg{Code: 0, Text: "a"})
	_ = e.View(80)
}

func TestStructList_LargeList(t *testing.T) {
	e := &structListEditor{}
	field := pkg.Field{
		ItemSchema: []pkg.FieldPart{{Name: "key", Type: "string"}, {Name: "val", Type: "string"}},
		Separator:  "=",
	}
	var lines []string
	for i := 0; i < 1000; i++ {
		lines = append(lines, "key"+strings.Repeat("x", 200)+"=val")
	}
	e.Init(field, strings.Join(lines, "\n"), testTheme())
	if len(e.items) != 1000 {
		t.Fatalf("expected 1000 items, got %d", len(e.items))
	}
	_ = e.View(80)
	_ = e.Height()
}

// ── listEditor ──────────────────────────────────────────────────────────────

func TestListEditor_EmptyInit(t *testing.T) {
	e := &listEditor{}
	e.Init(pkg.Field{}, "", testTheme())
	if len(e.items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(e.items))
	}
	v := e.Value()
	if v != "" {
		t.Fatalf("expected empty value, got %q", v)
	}
	_ = e.View(80)
	_ = e.Height()
}

func TestListEditor_SingleItem(t *testing.T) {
	e := &listEditor{}
	e.Init(pkg.Field{}, "solo", testTheme())
	if len(e.items) != 1 || e.items[0] != "solo" {
		t.Fatalf("expected [solo], got %v", e.items)
	}
}

func TestListEditor_WhitespaceOnlyItems(t *testing.T) {
	e := &listEditor{}
	e.Init(pkg.Field{}, "  \n  \n  ", testTheme())
	if len(e.items) != 0 {
		t.Fatalf("expected 0 items from whitespace-only, got %d: %v", len(e.items), e.items)
	}
}

func TestListEditor_DeleteOnEmpty(t *testing.T) {
	e := &listEditor{}
	e.Init(pkg.Field{}, "", testTheme())
	e.Update(tea.KeyPressMsg{Code: 0, Text: "d"})
	_ = e.View(80)
}

func TestListEditor_DoubleInit(t *testing.T) {
	th := testTheme()
	e := &listEditor{}
	e.Init(pkg.Field{}, "first\nsecond", th)
	e.Init(pkg.Field{}, "third", th)
	if len(e.items) != 1 || e.items[0] != "third" {
		t.Fatalf("double init did not reset: %v", e.items)
	}
}

func TestListEditor_PatternListCompletionEmpty(t *testing.T) {
	e := &listEditor{}
	field := pkg.Field{Widget: "patternlist", Options: []string{"opt1", "opt2"}}
	e.Init(field, "", testTheme())
	// start editing
	e.Update(tea.KeyPressMsg{Code: 0, Text: "a"})
	// filter with no input — should show all options
	e.filterCompletions()
	if len(e.completionFiltered) != 2 {
		t.Fatalf("expected 2 filtered options, got %d", len(e.completionFiltered))
	}
}

func TestListEditor_LargeList(t *testing.T) {
	var items []string
	for i := 0; i < 1000; i++ {
		items = append(items, "item"+strings.Repeat("y", 200))
	}
	e := &listEditor{}
	e.Init(pkg.Field{}, strings.Join(items, "\n"), testTheme())
	if len(e.items) != 1000 {
		t.Fatalf("expected 1000 items, got %d", len(e.items))
	}
	_ = e.View(80)
}

func TestMultiEditorShowsSelectAcceptHelper(t *testing.T) {
	e := &multiEditor{}
	e.Init(pkg.Field{Options: []string{"bold", "italic"}}, "bold", testTheme())

	got := stripANSI(e.View(80))
	if !strings.Contains(got, "␣:select") || !strings.Contains(got, "⏎:accept") {
		t.Fatalf("multi helper = %q, want select and accept hints", got)
	}
	if got, want := e.Height(), 3; got != want {
		t.Fatalf("multi height = %d, want %d", got, want)
	}
	if got := e.Interaction(); got != InteractionMulti {
		t.Fatalf("multi interaction = %v, want InteractionMulti", got)
	}
}

func TestFontEditor_FilterAcceptsJAndK(t *testing.T) {
	e := &fontEditor{}
	e.Init(pkg.Field{Widget: "font"}, "", testTheme())
	e.Update(FontsLoadedMsg{Fonts: []string{"JetBrains Mono", "Iosevka", "Noto Sans"}})
	e.cursor = 1

	e.Update(keyMsg("j"))
	e.Update(keyMsg("k"))

	if got := e.filter.Value(); got != "jk" {
		t.Fatalf("filter value = %q, want jk", got)
	}
}

func TestFontEditorMatchesFontconfigFamily(t *testing.T) {
	e := &fontEditor{}
	e.Init(pkg.Field{Widget: "font"}, "JetBrains Mono:size=11", testTheme())
	e.Update(FontsLoadedMsg{Fonts: []string{
		"A0", "A1", "A2", "A3", "A4", "A5", "A6", "A7",
		"JetBrains Mono", "Noto Sans",
	}})

	if got := e.cursor; got != 8 {
		t.Fatalf("cursor = %d, want current font index 8", got)
	}
	if got := e.viewOffset; got != 8 {
		t.Fatalf("viewOffset = %d, want selected font at top", got)
	}
}

func TestFontEditorPreservesFontconfigSuffixOnPickerSelect(t *testing.T) {
	e := &fontEditor{}
	e.Init(pkg.Field{Widget: "font"}, "JetBrains Mono:size=11", testTheme())
	e.Update(FontsLoadedMsg{Fonts: []string{"JetBrains Mono", "Iosevka", "Noto Sans"}})
	e.cursor = 1

	_, done, canceled := e.Update(keyMsg("enter"))
	if !done || canceled {
		t.Fatalf("enter done=%v canceled=%v, want committed", done, canceled)
	}
	if got, want := e.Value(), "Iosevka:size=11"; got != want {
		t.Fatalf("Value() = %q, want %q", got, want)
	}
}

func TestFontEditorFreeTextStartsFromRawFontconfigValue(t *testing.T) {
	e := &fontEditor{}
	e.Init(pkg.Field{Widget: "font"}, "JetBrains Mono:size=11", testTheme())
	e.Update(FontsLoadedMsg{Fonts: []string{"JetBrains Mono", "Iosevka"}})

	e.Update(keyMsg("tab"))

	if got, want := e.filter.Value(), "JetBrains Mono:size=11"; got != want {
		t.Fatalf("filter value = %q, want %q", got, want)
	}
}

// ── hookEditor ──────────────────────────────────────────────────────────────

func TestHookEditor_EmptyInit(t *testing.T) {
	e := &hookEditor{}
	e.Init(pkg.Field{}, "", testTheme())
	v := e.Value()
	if v != "[]" {
		t.Fatalf("expected '[]', got %q", v)
	}
	_ = e.View(80)
	_ = e.Height()
}

func TestHookEditor_EmptyArrayInit(t *testing.T) {
	e := &hookEditor{}
	e.Init(pkg.Field{}, "[]", testTheme())
	v := e.Value()
	if v != "[]" {
		t.Fatalf("expected '[]', got %q", v)
	}
}

func TestHookEditor_InvalidJSON(t *testing.T) {
	th := testTheme()
	cases := []string{`{invalid`, `null`, `"string"`, `42`, `{}`}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			e := &hookEditor{}
			e.Init(pkg.Field{}, input, th)
			v := e.Value()
			// should still produce valid JSON array
			if !json.Valid([]byte(v)) {
				t.Fatalf("input %q → invalid JSON output: %q", input, v)
			}
			_ = e.View(80)
		})
	}
}

func TestHookEditor_DeleteOnEmpty(t *testing.T) {
	e := &hookEditor{}
	e.Init(pkg.Field{}, "[]", testTheme())
	e.Update(tea.KeyPressMsg{Code: 0, Text: "d"})
	_ = e.View(80)
}

func TestHookEditor_DoubleInit(t *testing.T) {
	th := testTheme()
	e := &hookEditor{}
	e.Init(pkg.Field{}, `[{"matcher":"A","hooks":[{"type":"command","command":"echo a"}]}]`, th)
	e.Init(pkg.Field{}, `[{"matcher":"B","hooks":[{"type":"command","command":"echo b"}]}]`, th)
	if len(e.groups) != 1 || e.groups[0].Matcher != "B" {
		t.Fatalf("double init did not reset: %+v", e.groups)
	}
}

func TestHookEditor_GroupWithEmptyHooks(t *testing.T) {
	e := &hookEditor{}
	e.Init(pkg.Field{}, `[{"matcher":"test","hooks":[]}]`, testTheme())
	_ = e.View(80)
	v := e.Value()
	if !json.Valid([]byte(v)) {
		t.Fatalf("group with empty hooks produced invalid JSON: %q", v)
	}
}

func TestHookEditor_NegativeTimeout(t *testing.T) {
	e := &hookEditor{}
	input := `[{"matcher":"*","hooks":[{"type":"command","command":"test","timeout":-5}]}]`
	e.Init(pkg.Field{}, input, testTheme())
	v := e.Value()
	if !json.Valid([]byte(v)) {
		t.Fatalf("negative timeout produced invalid JSON: %q", v)
	}
}

func TestHookEditor_MatcherCompletion_EmptyOptions(t *testing.T) {
	e := &hookEditor{}
	e.Init(pkg.Field{Options: nil}, "[]", testTheme())
	// start add
	e.Update(tea.KeyPressMsg{Code: 0, Text: "a"})
	e.filterMatcherCompletions()
	// should not panic, no completions
	if e.matcherCompVisible {
		t.Fatalf("expected no completions with nil options")
	}
}

func TestHookEditor_LargeGroups(t *testing.T) {
	var groups []hookGroup
	for i := 0; i < 1000; i++ {
		groups = append(groups, hookGroup{
			Matcher: strings.Repeat("m", 200),
			Hooks:   []hookItem{{Type: "command", Command: strings.Repeat("c", 200)}},
		})
	}
	data, _ := json.Marshal(groups)
	e := &hookEditor{}
	e.Init(pkg.Field{}, string(data), testTheme())
	if len(e.groups) != 1000 {
		t.Fatalf("expected 1000 groups, got %d", len(e.groups))
	}
	_ = e.View(80)
}

// ── hookEditor filterMatcherCompletions aliasing ────────────────────────────

func TestHookEditor_FilterMatcherCompletions_Aliasing(t *testing.T) {
	// the filter reuses matcherFiltered[:0] as the backing array — this can
	// corrupt matcherOptions if they share the same backing array
	e := &hookEditor{}
	opts := []string{"Bash", "Python", "PreCommit", "PostCommit"}
	e.Init(pkg.Field{Options: opts}, "[]", testTheme())

	// start adding to activate editing mode
	e.Update(tea.KeyPressMsg{Code: 0, Text: "a"})

	// type a filter that matches a subset
	e.input.SetValue("commit")
	e.filterMatcherCompletions()

	// matcherFiltered should only have PreCommit and PostCommit
	if len(e.matcherFiltered) != 2 {
		t.Fatalf("expected 2 filtered results, got %d: %v", len(e.matcherFiltered), e.matcherFiltered)
	}

	// now check matcherOptions is not corrupted
	if len(e.matcherOptions) != 4 {
		t.Fatalf("matcherOptions corrupted after filter: expected 4, got %d: %v",
			len(e.matcherOptions), e.matcherOptions)
	}
	// verify original values preserved
	for i, want := range []string{"Bash", "Python", "PreCommit", "PostCommit"} {
		if e.matcherOptions[i] != want {
			t.Fatalf("matcherOptions[%d] = %q, want %q", i, e.matcherOptions[i], want)
		}
	}
}
