package theme

import "testing"

func TestTokyoNightMainAccentUsesOrange(t *testing.T) {
	if got := colorToHex(TokyoNight.Primary.Dark.TrueColor); got != "#ff9e64" {
		t.Fatalf("tokyonight primary dark = %s, want #ff9e64", got)
	}
	if got := colorToHex(TokyoNight.Accent.Dark.TrueColor); got != "#ff9e64" {
		t.Fatalf("tokyonight accent dark = %s, want #ff9e64", got)
	}
	if got := colorToHex(TokyoNight.BorderFocus.Dark.TrueColor); got != "#ff9e64" {
		t.Fatalf("tokyonight focused border dark = %s, want #ff9e64", got)
	}
	if got := colorToHex(TokyoNight.Primary.Dark.TrueColor); got == "#7aa2f7" || got == "#bb9af7" {
		t.Fatalf("tokyonight primary dark should not use blue or purple, got %s", got)
	}
}

func TestTokyoNightTextIsBrighter(t *testing.T) {
	if got := colorToHex(TokyoNight.Text.Dark.TrueColor); got != "#d5dcff" {
		t.Fatalf("tokyonight text dark = %s, want #d5dcff", got)
	}
}

func TestMonokaiPaletteIsAvailable(t *testing.T) {
	found := false
	for _, name := range PaletteNames() {
		if name == "monokai" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("PaletteNames missing monokai")
	}

	p := PaletteByName("monokai")
	if p.Name != "monokai" {
		t.Fatalf("PaletteByName(monokai) = %q", p.Name)
	}
	if got := colorToHex(p.Base.Dark.TrueColor); got != "#0c0c0c" {
		t.Fatalf("monokai base dark = %s, want #0c0c0c", got)
	}
	if got := colorToHex(p.Primary.Dark.TrueColor); got != "#9d65ff" {
		t.Fatalf("monokai primary dark = %s, want #9d65ff", got)
	}
	if got := colorToHex(p.Success.Dark.TrueColor); got != "#98e024" {
		t.Fatalf("monokai success dark = %s, want #98e024", got)
	}
}

func TestTomorrowNightPaletteIsAvailable(t *testing.T) {
	p := PaletteByName("tomorrow night")
	if p.Name != "tomorrow night" {
		t.Fatalf("PaletteByName(tomorrow night) = %q", p.Name)
	}
	if got := colorToHex(p.Base.Dark.TrueColor); got != "#2d2d2d" {
		t.Fatalf("tomorrow night base dark = %s, want #2d2d2d", got)
	}
	if got := colorToHex(p.Primary.Dark.TrueColor); got != "#6699cc" {
		t.Fatalf("tomorrow night primary dark = %s, want #6699cc", got)
	}
	if got := colorToHex(p.Error.Dark.TrueColor); got != "#f2777a" {
		t.Fatalf("tomorrow night error dark = %s, want #f2777a", got)
	}
}

func TestCatppuccinUsesMochaGhosttyColors(t *testing.T) {
	p := PaletteByName("catppuccin")
	if got := colorToHex(p.Base.Dark.TrueColor); got != "#1e1e2e" {
		t.Fatalf("catppuccin base dark = %s, want #1e1e2e", got)
	}
	if got := colorToHex(p.Primary.Dark.TrueColor); got != "#89b4fa" {
		t.Fatalf("catppuccin primary dark = %s, want #89b4fa", got)
	}
	if got := colorToHex(p.Muted.Dark.TrueColor); got != "#585b70" {
		t.Fatalf("catppuccin muted dark = %s, want #585b70", got)
	}
}

// new dark palettes must keep main (primary) hues distinct from the existing set.
func TestNewDarkPalettesHaveDistinctMainColors(t *testing.T) {
	want := map[string]string{
		"rose pine": "#ebbcba", // rose
		"solarized": "#2aa198", // cyan/teal
		"oxocarbon": "#ee5396", // magenta
	}
	for name, hex := range want {
		p := PaletteByName(name)
		if p.Name != name {
			t.Fatalf("PaletteByName(%q) = %q", name, p.Name)
		}
		if got := colorToHex(p.Primary.Dark.TrueColor); got != hex {
			t.Fatalf("%s primary dark = %s, want %s", name, got, hex)
		}
		// must not collide with an existing palette's main hue
		for _, other := range []string{"gh-dash", "catppuccin", "tomorrow night", "tokyonight", "gruvbox", "monokai", "nord"} {
			if colorToHex(PaletteByName(other).Primary.Dark.TrueColor) == hex {
				t.Fatalf("%s primary %s collides with %s", name, hex, other)
			}
		}
	}
}

func TestNewDarkPalettesUseDarkBase(t *testing.T) {
	bases := map[string]string{
		"rose pine": "#191724",
		"solarized": "#002b36",
		"oxocarbon": "#161616",
	}
	for name, hex := range bases {
		if got := PaletteByName(name).BaseHex(); got != hex {
			t.Fatalf("%s base = %s, want %s", name, got, hex)
		}
	}
}

func TestGhosttyThemeNameAliases(t *testing.T) {
	tests := map[string]string{
		"Catppuccin Mocha":        "catppuccin",
		"Monokai Remastered":      "monokai",
		"Tomorrow Night Eighties": "tomorrow night",
		"Rosé Pine":               "rose pine",
		"Solarized Dark":          "solarized",
	}

	for name, want := range tests {
		if got := PaletteByName(name).Name; got != want {
			t.Fatalf("PaletteByName(%q) = %q, want %q", name, got, want)
		}
	}
}
