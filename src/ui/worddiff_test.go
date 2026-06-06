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

// TestRenderWordDiffPreservesText asserts that styling never drops, adds, or
// reorders characters — the visible text must equal the input value.
func TestRenderWordDiffPreservesText(t *testing.T) {
	th := theme.NewTheme(theme.PaletteByName("catppuccin"))
	cases := []struct{ self, other string }{
		{"#1e1e1e", "#282828"},
		{"14", "12"},
		{"JetBrains Mono", "Fira Code"},
		{"café", "cafe"},
		{"value", "value"},
	}
	for _, c := range cases {
		for _, side := range []diffSide{diffRemoved, diffAdded} {
			got := ansiRE.ReplaceAllString(renderWordDiff(c.self, c.other, side, th), "")
			if got != c.self {
				t.Errorf("renderWordDiff(%q, %q, %d) visible text = %q, want %q",
					c.self, c.other, side, got, c.self)
			}
		}
	}
}

func TestDiffAffixes(t *testing.T) {
	tests := []struct {
		name   string
		a, b   string
		prefix int
		suffix int
	}{
		{"hex tail change", "#1e1e1e", "#282828", 1, 0},
		{"single digit", "12", "14", 1, 0},
		{"shared prefix and suffix", "12px", "14px", 1, 2},
		{"insertion", "12", "123", 2, 0},
		{"identical", "same", "same", 4, 0},
		{"no overlap", "abc", "xyz", 0, 0},
		{"empty other", "value", "", 0, 0},
		{"unicode", "café", "cafe", 3, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, s := diffAffixes(tt.a, tt.b)
			if p != tt.prefix || s != tt.suffix {
				t.Errorf("diffAffixes(%q, %q) = (%d, %d), want (%d, %d)",
					tt.a, tt.b, p, s, tt.prefix, tt.suffix)
			}
			// affixes must never overlap in either string
			ra, rb := []rune(tt.a), []rune(tt.b)
			if p+s > len(ra) || p+s > len(rb) {
				t.Errorf("overlapping affixes: prefix=%d suffix=%d len(a)=%d len(b)=%d",
					p, s, len(ra), len(rb))
			}
		})
	}
}
