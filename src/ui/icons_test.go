package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/getkonfi/konfi/pkg"
)

func TestIconCellFitsRequestedWidth(t *testing.T) {
	for _, tc := range []struct {
		name  string
		icon  string
		width int
	}{
		{name: "pads narrow icon", icon: "#", width: 3},
		{name: "keeps exact icon", icon: "[+]", width: 3},
		{name: "truncates wide icon", icon: "[+]", width: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := lipgloss.Width(iconCell(tc.icon, tc.width)); got != tc.width {
				t.Fatalf("iconCell(%q, %d) width = %d, want %d", tc.icon, tc.width, got, tc.width)
			}
		})
	}
}

func TestSidebarPlainIconUsesUnicodeSymbolWhenNerdFontDisabled(t *testing.T) {
	s := newSidebar(nil, testTheme())
	s.nerdFont = false

	if got := s.itemIcon(sidebarItem{name: "dconf", plainIcon: "⚙️"}); got != "⚙" {
		t.Fatalf("itemIcon(dconf) = %q, want symbol fallback", got)
	}
	if got := s.itemIcon(sidebarItem{name: "alacritty", plainIcon: "🖥️"}); got != "🖥" {
		t.Fatalf("itemIcon(alacritty) = %q, want symbol fallback", got)
	}
}

func TestRenderFieldRowsAlignPlainTypeIcons(t *testing.T) {
	c := newContent(testTheme())
	c.nerdFont = false
	c.schema = &pkg.Schema{Sections: []pkg.Section{{
		Name: "general",
		Fields: []pkg.Field{
			{Key: "text", Label: "Text", Type: "string", Default: "mono"},
			{Key: "size", Label: "Font Size", Type: "number", Default: "14"},
			{Key: "blocks", Label: "Blocks", Type: "string", Widget: "blocklist", Default: blocklistDefault(t)},
		},
	}}}
	c.buildFieldList()

	got := stripANSI(c.renderBody(90))
	labelOffsets := []int{
		displayOffset(t, got, "Text"),
		displayOffset(t, got, "Font Size"),
		displayOffset(t, got, "Blocks"),
	}
	if labelOffsets[0] != labelOffsets[1] || labelOffsets[1] != labelOffsets[2] {
		t.Fatalf("field labels are not aligned: offsets=%v\n%s", labelOffsets, got)
	}

	valueOffsets := []int{
		displayOffset(t, got, "mono"),
		displayOffset(t, got, "14"),
		displayOffset(t, got, "Host web"),
	}
	if valueOffsets[0] != valueOffsets[1] || valueOffsets[1] != valueOffsets[2] {
		t.Fatalf("field values are not aligned: offsets=%v\n%s", valueOffsets, got)
	}
}

func blocklistDefault(t *testing.T) string {
	t.Helper()

	return pkg.Encode(pkg.Parse([]byte("Host web\n    User git\n"), []string{"Host"}, nil))
}

func displayOffset(t *testing.T, haystack, needle string) int {
	t.Helper()

	for _, line := range strings.Split(haystack, "\n") {
		idx := strings.Index(line, needle)
		if idx >= 0 {
			return lipgloss.Width(line[:idx])
		}
	}
	t.Fatalf("missing %q in:\n%s", needle, haystack)
	return -1
}
