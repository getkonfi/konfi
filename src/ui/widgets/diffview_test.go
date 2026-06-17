package widgets

import (
	"strings"
	"testing"

	"github.com/getkonfi/konfi/theme"
)

func TestDiffViewPrefixesMultilineValues(t *testing.T) {
	th := theme.NewTheme(theme.PaletteByName("catppuccin"))
	d := NewDiffView(th)
	d.SetSize(100, 20)
	d.SetEntries([]PendingChange{{
		Section: "Host & Match Blocks",
		Key:     "Blocks",
		OldVal:  "Host github.com\n    User git",
		NewVal:  "Host github.com\n    User git\nHost konfi.local\n    User deploy",
	}})

	got := ansiRE.ReplaceAllString(d.View(), "")
	for _, want := range []string{
		"  - Host github.com",
		"  -     User git",
		"  + Host konfi.local",
		"  +     User deploy",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("diff view missing %q:\n%s", want, got)
		}
	}
}
