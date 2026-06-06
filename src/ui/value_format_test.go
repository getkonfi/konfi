package ui

import (
	"strings"
	"testing"

	"github.com/eminert/konfi/theme"
)

func TestStripKeyPrefix(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		key   string
		want  string
	}{
		{"copied form", "format = $all", "format", "$all"},
		{"no spaces", "format=$all", "format", "$all"},
		{"plain value untouched", "$all", "format", "$all"},
		{"non-matching key untouched", "other = x", "format", "other = x"},
		{"key as substring not stripped", "formatted", "format", "formatted"},
		{"value with equals preserved", "ctrl+a = goto_tab:1", "keybind", "ctrl+a = goto_tab:1"},
		{"empty key untouched", "a = b", "", "a = b"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := stripKeyPrefix(tc.value, tc.key); got != tc.want {
				t.Fatalf("stripKeyPrefix(%q, %q) = %q, want %q", tc.value, tc.key, got, tc.want)
			}
		})
	}
}

func TestSingleLine(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"$os$username", "$os$username"}, // clean value untouched
		{"a\nb", "a\\nb"},                // real newline escaped
		{"a\tb", "a\\tb"},                // tab escaped
		{"a\r\nb", "a\\nb"},              // crlf collapses to one escape
	} {
		if got := singleLine(tc.in); got != tc.want {
			t.Fatalf("singleLine(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLowContrast(t *testing.T) {
	const bg = "#1e1e2e" // catppuccin base
	if !theme.LowContrast(bg, bg) {
		t.Fatal("identical color/background should be low contrast")
	}
	if !theme.LowContrast("#222232", bg) {
		t.Fatal("near-background color should be low contrast")
	}
	if theme.LowContrast("#ffffff", bg) {
		t.Fatal("white on dark base should not be low contrast")
	}
}

// colorValue adds a background backdrop only when the tint is too close to bg.
func TestColorValueContrastBackdrop(t *testing.T) {
	const bg = "#1e1e2e"

	nearBg := theme.ColorValue("#1f1f30", bg)
	if !strings.Contains(nearBg, "48;2") {
		t.Fatalf("near-background color should get a contrast backdrop, got %q", nearBg)
	}

	readable := theme.ColorValue("#ffffff", bg)
	if strings.Contains(readable, "48;2") {
		t.Fatalf("readable color should not get a backdrop, got %q", readable)
	}

	// no ## marker in either case
	if strings.Contains(stripANSI(nearBg), "##") || strings.Contains(stripANSI(readable), "##") {
		t.Fatal("color value should not contain a ## marker")
	}
}
