package editors

import (
	"strings"
	"testing"

	"github.com/eminert/konfi/pkg"
)

func TestNumberEditorShowsArrowAdjustHint(t *testing.T) {
	e := &numberEditor{}
	e.Init(pkg.Field{Type: "number"}, "10", testTheme())

	got := stripANSI(e.View(80))
	if !strings.Contains(got, "↑↓ adjust") {
		t.Fatalf("number editor view missing arrow adjust hint:\n%s", got)
	}

	inline := stripANSI(e.InlineView(80))
	if !strings.Contains(inline, "↑↓ adjust") {
		t.Fatalf("number editor inline view missing arrow adjust hint:\n%s", inline)
	}
}

func TestNumberEditorCombinesRangeAndArrowHint(t *testing.T) {
	minVal, maxVal := 0.0, 255.0
	e := &numberEditor{}
	e.Init(pkg.Field{Type: "number", Min: &minVal, Max: &maxVal}, "10", testTheme())

	got := stripANSI(e.View(80))
	if !strings.Contains(got, "(0 — 255)") {
		t.Fatalf("number editor view missing range hint:\n%s", got)
	}
	if !strings.Contains(got, "↑↓ adjust") {
		t.Fatalf("number editor view missing arrow adjust hint:\n%s", got)
	}
}

func TestNumberEditorStepsEmptyValueFromZero(t *testing.T) {
	e := &numberEditor{}
	e.Init(pkg.Field{Type: "number"}, "", testTheme())

	_, done, canceled := e.Update(keyMsg("up"))
	if done || canceled {
		t.Fatalf("up should keep editor open: done=%v canceled=%v", done, canceled)
	}
	if got := e.input.Value(); got != "1" {
		t.Fatalf("up from empty value = %q, want 1", got)
	}

	e.input.SetValue("")
	e.Update(keyMsg("down"))
	if got := e.input.Value(); got != "-1" {
		t.Fatalf("down from empty value = %q, want -1", got)
	}
}
