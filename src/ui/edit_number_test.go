package ui

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
