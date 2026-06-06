package ui

import (
	"strings"

	"github.com/eminert/konfi/theme"
)

// diffSide selects the removed (old) or added (new) styling for a word diff.
type diffSide int

const (
	diffRemoved diffSide = iota
	diffAdded
)

// renderWordDiff renders self with its differing middle emphasized, like a git
// inline diff. other is the opposing value, used only to locate the shared
// prefix/suffix so that just the changed run is highlighted. self and other are
// expected to differ; if they don't, the whole value renders in the base style.
func renderWordDiff(self, other string, side diffSide, th *theme.Theme) string {
	base := th.Error
	emph := th.Error.Bold(true).Background(th.Palette.Surface)
	if side == diffAdded {
		base = th.Success
		emph = th.Success.Bold(true).Background(th.Palette.Surface)
	}

	prefix, suffix := diffAffixes(self, other)
	r := []rune(self)
	mid := r[prefix : len(r)-suffix]
	if len(mid) == 0 {
		return base.Render(self)
	}

	var b strings.Builder
	b.WriteString(base.Render(string(r[:prefix])))
	b.WriteString(emph.Render(string(mid)))
	b.WriteString(base.Render(string(r[len(r)-suffix:])))
	return b.String()
}

// diffAffixes returns the lengths (in runes) of the longest common prefix and
// suffix shared by a and b. The two ranges never overlap in either string.
func diffAffixes(a, b string) (prefix, suffix int) {
	ra, rb := []rune(a), []rune(b)
	n := min(len(ra), len(rb))

	for prefix < n && ra[prefix] == rb[prefix] {
		prefix++
	}
	for suffix < n-prefix && ra[len(ra)-1-suffix] == rb[len(rb)-1-suffix] {
		suffix++
	}
	return prefix, suffix
}
