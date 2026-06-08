package ui

import (
	"reflect"
	"testing"
)

// TestSidebarGroupsNotInstalled verifies refilter orders entries as
// home → installed → not-installed → system, so uninstalled apps form their own
// section regardless of registry order, while preserving order within a group.
func TestSidebarGroupsNotInstalled(t *testing.T) {
	s := newSidebar([]sidebarItem{
		{name: "home", installed: true, home: true},
		{name: "ghostty", installed: true},
		{name: "gnome", installed: false},
		{name: "starship", installed: true},
		{name: "kitty", installed: false},
		{name: "konfi", installed: true, system: true},
	}, testTheme())

	var order []string
	for _, idx := range s.filtered {
		order = append(order, s.items[idx].name)
	}
	want := []string{"home", "ghostty", "starship", "gnome", "kitty", "konfi"}
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
