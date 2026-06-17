package ui

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	"github.com/getkonfi/konfi/theme"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"github.com/charmbracelet/colorprofile"
)

func TestStatusbarQuietHelpersReadableAcrossPalettes(t *testing.T) {
	withTrueColor(t)

	for _, dark := range []bool{false, true} {
		theme.SetTerminalBackgroundDark(dark)
		for i := range theme.Palettes {
			palette := theme.Palettes[i]
			th := theme.NewTheme(&palette)
			s := newStatusbar(th)
			s.width = 100

			keyStyle, hintStyle, themeStyle := s.hintStyles(statusQuiet)
			prefix := fmt.Sprintf("%s dark=%v", palette.Name, dark)
			assertReadableStyle(t, prefix+" statusbar", th.Statusbar)
			assertReadableStyle(t, prefix+" quiet key", keyStyle)
			assertReadableStyle(t, prefix+" quiet hint", hintStyle)
			assertReadableStyle(t, prefix+" quiet theme", themeStyle)
		}
	}
}

func TestStatusbarToneStylesReadableAcrossPalettes(t *testing.T) {
	withTrueColor(t)

	tones := []struct {
		name string
		tone statusTone
	}{
		{"dirty", statusDirty},
		{"preview", statusPreview},
		{"saving", statusSaving},
		{"saved", statusSaved},
		{"error", statusError},
	}

	for _, dark := range []bool{false, true} {
		theme.SetTerminalBackgroundDark(dark)
		for i := range theme.Palettes {
			palette := theme.Palettes[i]
			th := theme.NewTheme(&palette)
			s := newStatusbar(th)
			s.width = 100

			prefix := fmt.Sprintf("%s dark=%v", palette.Name, dark)
			assertReadableStyle(t, prefix+" edit badge", s.editBadge)
			assertReadableStyle(t, prefix+" search badge", s.searchBadge)

			for _, tc := range tones {
				keyStyle, hintStyle, themeStyle := s.hintStyles(tc.tone)
				assertReadableStyle(t, prefix+" "+tc.name+" band", s.bandStyle(tc.tone))
				assertReadableStyle(t, prefix+" "+tc.name+" signal", s.signalStyle(tc.tone))
				assertReadableStyle(t, prefix+" "+tc.name+" key", keyStyle)
				assertReadableStyle(t, prefix+" "+tc.name+" hint", hintStyle)
				assertReadableStyle(t, prefix+" "+tc.name+" theme", themeStyle)
			}
		}
	}
}

func TestStatusbarToneDoesNotTreatUnsavedAsSaved(t *testing.T) {
	s := newStatusbar(testTheme())

	s.status = "unsaved changes - q to quit, ctrl+s to save"
	if got := s.tone(); got != statusDirty {
		t.Fatalf("unsaved confirmation tone = %v, want dirty", got)
	}

	s.status = "no unsaved changes"
	if got := s.tone(); got != statusQuiet {
		t.Fatalf("no-unsaved feedback tone = %v, want quiet", got)
	}

	s.status = "saved"
	if got := s.tone(); got != statusSaved {
		t.Fatalf("saved tone = %v, want saved", got)
	}
}

func TestStatusbarDirtyViewShowsLowercaseUnsavedAppNameOnly(t *testing.T) {
	s := newStatusbar(testTheme())
	s.width = 100
	s.appName = "ghostty"
	s.dirtyApps = []string{"ghostty", "starship"}
	s.changeCount = 1
	s.undoCount = 2
	s.status = "unsaved changes — q to quit, ctrl+s to save"

	got := stripANSI(s.View())
	if !strings.Contains(got, "unsaved") {
		t.Fatalf("dirty status missing lowercase unsaved marker:\n%s", got)
	}
	if strings.Contains(got, "UNSAVED") {
		t.Fatalf("dirty status rendered uppercase marker:\n%s", got)
	}
	if !strings.Contains(got, "ghostty, starship") {
		t.Fatalf("dirty status missing app names:\n%s", got)
	}
	for _, unwanted := range []string{"unsaved changes", "ctrl+s", "q to quit", "↩", "1 unsaved change"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("dirty status should not render %q:\n%s", unwanted, got)
		}
	}
}

func TestStatusbarDirtyHintsKeepQuietBackground(t *testing.T) {
	s := newStatusbar(testTheme())

	if !sameColor(s.lineStyle(statusDirty).GetBackground(), s.theme.Palette.Base) {
		t.Fatal("dirty status line should keep the base statusbar background")
	}

	keyStyle, hintStyle, themeStyle := s.hintStyles(statusDirty)
	if !sameColor(keyStyle.GetBackground(), s.theme.Palette.Surface) {
		t.Fatal("dirty shortcut key background should stay neutral")
	}
	if !sameColor(hintStyle.GetBackground(), s.theme.Palette.Base) {
		t.Fatal("dirty shortcut label background should stay neutral")
	}
	if !sameColor(themeStyle.GetBackground(), s.theme.Palette.Base) {
		t.Fatal("dirty theme badge background should stay neutral")
	}
}

func withTrueColor(t *testing.T) {
	t.Helper()
	oldDark := compat.HasDarkBackground
	oldProfile := compat.Profile
	theme.SetTerminalColorProfile(colorprofile.TrueColor)
	t.Cleanup(func() {
		compat.HasDarkBackground = oldDark
		compat.Profile = oldProfile
	})
}

func assertReadableStyle(t *testing.T, name string, style lipgloss.Style) {
	t.Helper()
	ratio := theme.ContrastRatio(style.GetForeground(), style.GetBackground())
	if ratio < 4.5 {
		t.Fatalf("%s contrast = %.2f, want >= 4.50", name, ratio)
	}
}

func sameColor(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}
