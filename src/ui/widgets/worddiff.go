package widgets

import (
	"strings"

	"github.com/getkonfi/konfi/theme"
)

// DiffSide selects the removed (old) or added (new) styling for a word diff.
type DiffSide int

const (
	DiffRemoved DiffSide = iota
	DiffAdded
)

// RenderWordDiff renders self with its changed run emphasized, like a git inline
// diff. other is the opposing value, used only to locate the shared leading run;
// everything from the first difference through the end of self is highlighted.
// self and other are expected to differ; if they don't, the whole value renders
// in the base style.
func RenderWordDiff(self, other string, side DiffSide, th *theme.Theme) string {
	base := th.Error
	emph := th.Error.Bold(true).Background(th.Palette.Surface)
	if side == DiffAdded {
		base = th.Success
		emph = th.Success.Bold(true).Background(th.Palette.Surface)
	}

	prefix := commonPrefix(self, other)
	r := []rune(self)
	if prefix >= len(r) {
		return base.Render(self)
	}

	var b strings.Builder
	b.WriteString(base.Render(string(r[:prefix])))
	b.WriteString(emph.Render(string(r[prefix:])))
	return b.String()
}

// commonPrefix returns the length (in runes) of the longest common leading run
// shared by a and b.
func commonPrefix(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	n := min(len(ra), len(rb))
	i := 0
	for i < n && ra[i] == rb[i] {
		i++
	}
	return i
}
