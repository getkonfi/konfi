package ui

import (
	"reflect"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestSidebarGroupsNotInstalled verifies refilter orders entries as
// home → installed → not-installed → system, with each group sorted by name.
func TestSidebarGroupsNotInstalled(t *testing.T) {
	s := newSidebar([]sidebarItem{
		{name: "home", installed: true, home: true},
		{name: "starship", installed: true},
		{name: "kitty", installed: false},
		{name: "ghostty", installed: true},
		{name: "gnome", installed: false},
		{name: "konfi", installed: true, system: true},
		{name: "about", installed: true, system: true},
	}, testTheme())

	var order []string
	for _, idx := range s.filtered {
		order = append(order, s.items[idx].name)
	}
	want := []string{"home", "ghostty", "starship", "gnome", "kitty", "about", "konfi"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("sidebar order = %v, want %v", order, want)
	}
}

// TestSidebarNotInstalledSelectionIndex guards that grouping uninstalled apps
// into their own section does not corrupt the konfable index: emitSelection
// derives it from the stable items position, not the reordered filtered slice.
func TestSidebarNotInstalledSelectionIndex(t *testing.T) {
	s := newSidebar([]sidebarItem{
		{name: "home", installed: true, home: true},
		{name: "ghostty", installed: true}, // items[1] → konfable 0
		{name: "gnome", installed: false},  // items[2] → konfable 1
		{name: "starship", installed: true},
	}, testTheme())

	// locate gnome in the (reordered) filtered list and select it
	target := -1
	for fi, idx := range s.filtered {
		if s.items[idx].name == "gnome" {
			target = fi
			break
		}
	}
	if target < 0 {
		t.Fatal("gnome not present in filtered list")
	}
	s.cursor = target

	_, cmd := s.selectCurrent()
	msg, ok := cmd().(AppSelectedMsg)
	if !ok {
		t.Fatalf("selectCurrent emitted %T, want AppSelectedMsg", cmd())
	}
	if msg.Index != 1 || msg.Name != "gnome" {
		t.Fatalf("selection = {Index:%d Name:%q}, want {Index:1 Name:\"gnome\"}", msg.Index, msg.Name)
	}
}

func TestSidebarNotInstalledHeaderIsLowercase(t *testing.T) {
	s := newSidebar([]sidebarItem{
		{name: "home", installed: true, home: true},
		{name: "ghostty", installed: true},
		{name: "alacritty", installed: false},
		{name: "powerlevel10k", installed: false},
	}, testTheme())
	s.width = 32
	s.height = 12
	s.focused = true

	got := stripANSI(s.View())
	if strings.Contains(got, "NOT INSTALLED") {
		t.Fatalf("sidebar still renders uppercase not-installed header:\n%s", got)
	}
	if !strings.Contains(got, "not installed") {
		t.Fatalf("sidebar missing lowercase not-installed header:\n%s", got)
	}
	if !strings.Contains(got, "alacritty") || !strings.Contains(got, "powerlevel10k") {
		t.Fatalf("sidebar missing expected uninstalled apps:\n%s", got)
	}
}

func TestSidebarSystemHeaderHidden(t *testing.T) {
	s := newSidebar([]sidebarItem{
		{name: "home", installed: true, home: true},
		{name: "ghostty", installed: true},
		{name: "konfi", installed: true, system: true},
	}, testTheme())
	s.width = 32
	s.height = 12
	s.focused = true

	got := stripANSI(s.View())
	if strings.Contains(got, "SYSTEM") {
		t.Fatalf("sidebar rendered system header:\n%s", got)
	}
	if !strings.Contains(got, "konfi") {
		t.Fatalf("sidebar missing system item:\n%s", got)
	}
}

func TestSidebarPlainIconNamesAlign(t *testing.T) {
	s := newSidebar([]sidebarItem{}, testTheme())
	s.width = 32
	s.height = 8
	s.focused = true
	s.nerdFont = false

	oneCell := stripANSI(s.renderItem(sidebarItem{name: "dconf", plainIcon: "⚙️", installed: true}, false, 24))
	twoCell := stripANSI(s.renderItem(sidebarItem{name: "pacman", plainIcon: "📦", installed: true}, false, 24))

	got := prefixWidthBefore(oneCell, "dconf")
	want := prefixWidthBefore(twoCell, "pacman")
	if got != want {
		t.Fatalf("plain icon name prefix widths = %d and %d; lines:\n%q\n%q", got, want, oneCell, twoCell)
	}
}

func prefixWidthBefore(line, marker string) int {
	idx := strings.Index(line, marker)
	if idx < 0 {
		return -1
	}
	return lipgloss.Width(line[:idx])
}
