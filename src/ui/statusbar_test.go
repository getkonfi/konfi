package ui

import (
	"fmt"
	"testing"

	"github.com/eminert/konfi/theme"

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
