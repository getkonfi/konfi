package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestRenderDashboardOrdersAppsAlphabeticallyWithinGroups(t *testing.T) {
	c := newContent(testTheme())
	c.dashboardApps = []dashboardApp{
		{name: "starship", installed: true, configuredCount: 10, totalFields: 120},
		{name: "kitty", installed: false, totalFields: 300, minAppVersion: "0.32.0", maxAppVersion: "1.0.0"},
		{name: "ghostty", installed: true, configuredCount: 1, totalFields: 12},
		{name: "fuzzel", installed: false, totalFields: 1},
		{name: "gnome", installed: false, totalFields: 9, maxAppVersion: "47"},
		{name: "konfi", installed: true},
	}

	got := stripANSI(c.renderDashboard(100))

	assertBefore(t, got, "ghostty", "starship")
	assertBefore(t, got, "fuzzel", "gnome")
	assertBefore(t, got, "gnome", "kitty")
	for _, want := range []string{"12 fields", "120 fields", "1 field", "9 fields", "300 fields"} {
		if !strings.Contains(got, want) {
			t.Fatalf("dashboard missing field count %q:\n%s", want, got)
		}
	}
	fuzzelLine := lineContaining(t, got, "fuzzel")
	gnomeLine := lineContaining(t, got, "gnome")
	kittyLine := lineContaining(t, got, "kitty")
	fuzzelFieldIdx := displayIndex(fuzzelLine, "field")
	gnomeFieldIdx := displayIndex(gnomeLine, "field")
	kittyFieldIdx := displayIndex(kittyLine, "field")
	if fuzzelFieldIdx != gnomeFieldIdx || gnomeFieldIdx != kittyFieldIdx {
		t.Fatalf("not-detected field labels are not aligned:\n%s\n%s\n%s", fuzzelLine, gnomeLine, kittyLine)
	}
}

func displayIndex(haystack, needle string) int {
	idx := strings.Index(haystack, needle)
	if idx < 0 {
		return -1
	}
	return lipgloss.Width(haystack[:idx])
}

func TestFieldCountLabelPluralizes(t *testing.T) {
	if got := fieldCountLabel(1, 3); got != "  1 field" {
		t.Fatalf("fieldCountLabel(1, 3) = %q, want %q", got, "  1 field")
	}
	if got := fieldCountLabel(2, 3); got != "  2 fields" {
		t.Fatalf("fieldCountLabel(2, 3) = %q, want %q", got, "  2 fields")
	}
}

func assertBefore(t *testing.T, haystack, first, second string) {
	t.Helper()

	firstIdx := strings.Index(haystack, first)
	if firstIdx < 0 {
		t.Fatalf("dashboard missing %q:\n%s", first, haystack)
	}
	secondIdx := strings.Index(haystack, second)
	if secondIdx < 0 {
		t.Fatalf("dashboard missing %q:\n%s", second, haystack)
	}
	if firstIdx > secondIdx {
		t.Fatalf("dashboard rendered %q after %q:\n%s", first, second, haystack)
	}
}

func lineContaining(t *testing.T, haystack, needle string) string {
	t.Helper()

	for _, line := range strings.Split(haystack, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	t.Fatalf("dashboard missing line containing %q:\n%s", needle, haystack)
	return ""
}
