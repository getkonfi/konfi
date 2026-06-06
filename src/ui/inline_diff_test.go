package ui

import (
	"testing"

	"github.com/eminert/konfi/theme"
)

// TestRenderInlineDiff asserts the changed-only (tab) value renders as
// "old → new", with ∅ for absent sides, and never loses characters.
func TestRenderInlineDiff(t *testing.T) {
	c := &content{theme: theme.NewTheme(theme.PaletteByName("catppuccin"))}
	cases := []struct {
		oldVal string
		hadOld bool
		newVal string
		hasNew bool
		want   string
	}{
		{"12", true, "14", true, "12 → 14"},
		{"#1e1e1e", true, "#282828", true, "#1e1e1e → #282828"},
		{"", false, "enabled", true, "∅ → enabled"},
		{"removed", true, "", false, "removed → ∅"},
	}
	for _, tc := range cases {
		got := ansiRE.ReplaceAllString(c.renderInlineDiff(tc.oldVal, tc.hadOld, tc.newVal, tc.hasNew, 60), "")
		if got != tc.want {
			t.Errorf("renderInlineDiff(%q,%v,%q,%v) = %q, want %q",
				tc.oldVal, tc.hadOld, tc.newVal, tc.hasNew, got, tc.want)
		}
	}
}
