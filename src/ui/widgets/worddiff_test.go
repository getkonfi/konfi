package widgets

import (
	"regexp"
	"strings"
	"testing"

	"github.com/eminert/konfi/theme"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

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
		for _, side := range []DiffSide{DiffRemoved, DiffAdded} {
			got := ansiRE.ReplaceAllString(RenderWordDiff(c.self, c.other, side, th), "")
			if got != c.self {
				t.Errorf("RenderWordDiff(%q, %q, %d) visible text = %q, want %q",
					c.self, c.other, side, got, c.self)
			}
		}
	}
}

func TestCommonPrefix(t *testing.T) {
	tests := []struct {
		name   string
		a, b   string
		prefix int
	}{
		{"hex tail change", "#1e1e1e", "#282828", 1},
		{"single digit", "12", "14", 1},
		{"shared prefix only", "12px", "14px", 1},
		{"insertion", "12", "123", 2},
		{"identical", "same", "same", 4},
		{"no overlap", "abc", "xyz", 0},
		{"empty other", "value", "", 0},
		{"unicode", "café", "cafe", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := commonPrefix(tt.a, tt.b)
			if p != tt.prefix {
				t.Errorf("commonPrefix(%q, %q) = %d, want %d", tt.a, tt.b, p, tt.prefix)
			}
		})
	}
}

// TestRenderWordDiffHighlightsLastChar guards the policy that the changed run
// extends to the end of the value: even when old and new share a trailing
// character, the final rune must carry the emphasis background.
func TestRenderWordDiffHighlightsLastChar(t *testing.T) {
	th := theme.NewTheme(theme.PaletteByName("catppuccin"))
	changed := []struct{ self, other string }{
		{"false", "true"},      // shared trailing "e"
		{"#1e1e2e", "#1e1e1e"}, // shared trailing "e"
		{"12px", "10px"},       // shared trailing "px"
		{"2000", "1000"},       // shared trailing "000"
	}
	for _, c := range changed {
		for _, side := range []DiffSide{DiffRemoved, DiffAdded} {
			if got := RenderWordDiff(c.self, c.other, side, th); !bgAtLastRune(got) {
				t.Errorf("RenderWordDiff(%q, %q, %d): last rune lost its highlight", c.self, c.other, side)
			}
		}
	}
	// identical values carry no emphasis at all
	if bgAtLastRune(RenderWordDiff("same", "same", DiffAdded, th)) {
		t.Errorf("identical values should not be highlighted")
	}
}

var sgrRE = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

// bgAtLastRune reports whether a background SGR (48;…) is active at the final
// visible rune of a rendered string.
func bgAtLastRune(s string) bool {
	bg, last := false, false
	for i := 0; i < len(s); {
		if loc := sgrRE.FindStringIndex(s[i:]); loc != nil && loc[0] == 0 {
			p := sgrRE.FindStringSubmatch(s[i:])[1]
			switch {
			case p == "" || p == "0":
				bg = false
			case strings.Contains(p, "48;"):
				bg = true
			}
			i += loc[1]
			continue
		}
		r := []rune(s[i:])[0]
		last = bg
		i += len(string(r))
	}
	return last
}
