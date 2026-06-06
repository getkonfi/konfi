package theme

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// Palette defines a complete set of colors for theming.
// uses compat.CompleteAdaptiveColor for light/dark terminal support.
type Palette struct {
	Name string

	// primary surfaces
	Base    compat.CompleteAdaptiveColor
	Surface compat.CompleteAdaptiveColor
	Overlay compat.CompleteAdaptiveColor

	// text
	Text    compat.CompleteAdaptiveColor
	Subtext compat.CompleteAdaptiveColor
	Muted   compat.CompleteAdaptiveColor

	// accents
	Primary   compat.CompleteAdaptiveColor
	Secondary compat.CompleteAdaptiveColor
	Accent    compat.CompleteAdaptiveColor

	// semantic
	Success compat.CompleteAdaptiveColor
	Warning compat.CompleteAdaptiveColor
	Error   compat.CompleteAdaptiveColor

	// borders
	Border      compat.CompleteAdaptiveColor
	BorderFocus compat.CompleteAdaptiveColor
}

// built-in palettes
var Palettes = []Palette{GhDash, Catppuccin, TokyoNight, Gruvbox, Nord}

// PaletteByName returns the palette with the given name, or Catppuccin as default.
func PaletteByName(name string) *Palette {
	for i := range Palettes {
		if Palettes[i].Name == name {
			return &Palettes[i]
		}
	}
	return &Palettes[0]
}

// PaletteHex is a named color extracted from a palette.
type PaletteHex struct {
	Name string
	Hex  string
}

// colorToHex converts a color.Color to a "#rrggbb" hex string.
func colorToHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

// BaseHex returns the base background color as "#rrggbb" (dark-mode TrueColor,
// matching Hexes). used to detect color values that would render unreadably
// close to the background.
func (p Palette) BaseHex() string {
	return colorToHex(p.Base.Dark.TrueColor)
}

// Hexes returns the palette's accent and semantic colors as hex strings.
// uses dark-mode TrueColor values.
func (p Palette) Hexes() []PaletteHex {
	return []PaletteHex{
		{"base", colorToHex(p.Base.Dark.TrueColor)},
		{"surface", colorToHex(p.Surface.Dark.TrueColor)},
		{"overlay", colorToHex(p.Overlay.Dark.TrueColor)},
		{"text", colorToHex(p.Text.Dark.TrueColor)},
		{"muted", colorToHex(p.Muted.Dark.TrueColor)},
		{"primary", colorToHex(p.Primary.Dark.TrueColor)},
		{"secondary", colorToHex(p.Secondary.Dark.TrueColor)},
		{"accent", colorToHex(p.Accent.Dark.TrueColor)},
		{"success", colorToHex(p.Success.Dark.TrueColor)},
		{"warning", colorToHex(p.Warning.Dark.TrueColor)},
		{"error", colorToHex(p.Error.Dark.TrueColor)},
	}
}

// PaletteNames returns all available palette names.
func PaletteNames() []string {
	names := make([]string, len(Palettes))
	for i := range Palettes {
		names[i] = Palettes[i].Name
	}
	return names
}

// cc is shorthand for constructing a compat.CompleteColor from hex strings.
func cc(trueColor, ansi256, ansi string) compat.CompleteColor {
	return compat.CompleteColor{
		TrueColor: lipgloss.Color(trueColor),
		ANSI256:   lipgloss.Color(ansi256),
		ANSI:      lipgloss.Color(ansi),
	}
}

// cac is shorthand for constructing a compat.CompleteAdaptiveColor.
func cac(light, dark compat.CompleteColor) compat.CompleteAdaptiveColor {
	return compat.CompleteAdaptiveColor{Light: light, Dark: dark}
}

var Catppuccin = Palette{
	Name:        "catppuccin",
	Base:        cac(cc("#eff1f5", "255", "15"), cc("#1e1e2e", "234", "0")),
	Surface:     cac(cc("#ccd0da", "252", "7"), cc("#313244", "236", "8")),
	Overlay:     cac(cc("#bcc0cc", "250", "7"), cc("#45475a", "238", "8")),
	Text:        cac(cc("#4c4f69", "239", "0"), cc("#cdd6f4", "254", "15")),
	Subtext:     cac(cc("#6c6f85", "242", "8"), cc("#bac2de", "252", "7")),
	Muted:       cac(cc("#9ca0b0", "246", "8"), cc("#6c7086", "242", "8")),
	Primary:     cac(cc("#8839ef", "129", "5"), cc("#cba6f7", "183", "13")),
	Secondary:   cac(cc("#1e66f5", "33", "4"), cc("#89b4fa", "111", "12")),
	Accent:      cac(cc("#179299", "30", "6"), cc("#94e2d5", "115", "14")),
	Success:     cac(cc("#40a02b", "34", "2"), cc("#a6e3a1", "114", "10")),
	Warning:     cac(cc("#df8e1d", "172", "3"), cc("#f9e2af", "223", "11")),
	Error:       cac(cc("#d20f39", "160", "1"), cc("#f38ba8", "211", "9")),
	Border:      cac(cc("#9ca0b0", "246", "8"), cc("#585b70", "240", "8")),
	BorderFocus: cac(cc("#8839ef", "129", "5"), cc("#cba6f7", "183", "13")),
}

var GhDash = Palette{
	Name:        "gh-dash",
	Base:        cac(cc("#f6f8fa", "255", "15"), cc("#0d1117", "233", "0")),
	Surface:     cac(cc("#eef2f7", "254", "7"), cc("#161b22", "235", "8")),
	Overlay:     cac(cc("#d8dee4", "252", "7"), cc("#21262d", "236", "8")),
	Text:        cac(cc("#24292f", "236", "0"), cc("#c9d1d9", "252", "7")),
	Subtext:     cac(cc("#57606a", "241", "8"), cc("#8b949e", "246", "8")),
	Muted:       cac(cc("#6e7781", "243", "8"), cc("#6e7681", "242", "8")),
	Primary:     cac(cc("#0969da", "25", "4"), cc("#58a6ff", "75", "12")),
	Secondary:   cac(cc("#1f883d", "28", "2"), cc("#3fb950", "71", "10")),
	Accent:      cac(cc("#0a7ea4", "31", "6"), cc("#79c0ff", "117", "14")),
	Success:     cac(cc("#1f883d", "28", "2"), cc("#3fb950", "71", "10")),
	Warning:     cac(cc("#9a6700", "94", "3"), cc("#d29922", "178", "11")),
	Error:       cac(cc("#cf222e", "160", "1"), cc("#f85149", "203", "9")),
	Border:      cac(cc("#d0d7de", "252", "7"), cc("#30363d", "238", "8")),
	BorderFocus: cac(cc("#0969da", "25", "4"), cc("#58a6ff", "75", "12")),
}

var TokyoNight = Palette{
	Name:        "tokyonight",
	Base:        cac(cc("#d5d6db", "253", "15"), cc("#1a1b26", "234", "0")),
	Surface:     cac(cc("#c4c5cb", "251", "7"), cc("#24283b", "235", "8")),
	Overlay:     cac(cc("#b4b5bb", "249", "7"), cc("#414868", "238", "8")),
	Text:        cac(cc("#343b58", "237", "0"), cc("#c0caf5", "253", "15")),
	Subtext:     cac(cc("#4e5579", "240", "8"), cc("#a9b1d6", "249", "7")),
	Muted:       cac(cc("#6e7191", "243", "8"), cc("#565f89", "242", "8")),
	Primary:     cac(cc("#7830f0", "93", "5"), cc("#bb9af7", "141", "13")),
	Secondary:   cac(cc("#2e7de9", "33", "4"), cc("#7aa2f7", "111", "12")),
	Accent:      cac(cc("#007197", "30", "6"), cc("#7dcfff", "117", "14")),
	Success:     cac(cc("#587539", "64", "2"), cc("#9ece6a", "149", "10")),
	Warning:     cac(cc("#8c6c3e", "136", "3"), cc("#e0af68", "179", "11")),
	Error:       cac(cc("#c64343", "160", "1"), cc("#f7768e", "204", "9")),
	Border:      cac(cc("#9ca0b0", "246", "8"), cc("#414868", "238", "8")),
	BorderFocus: cac(cc("#7830f0", "93", "5"), cc("#bb9af7", "141", "13")),
}

var Gruvbox = Palette{
	Name:        "gruvbox",
	Base:        cac(cc("#fbf1c7", "230", "15"), cc("#282828", "235", "0")),
	Surface:     cac(cc("#ebdbb2", "223", "7"), cc("#3c3836", "237", "8")),
	Overlay:     cac(cc("#d5c4a1", "187", "7"), cc("#504945", "239", "8")),
	Text:        cac(cc("#3c3836", "237", "0"), cc("#ebdbb2", "223", "15")),
	Subtext:     cac(cc("#504945", "239", "8"), cc("#d5c4a1", "187", "7")),
	Muted:       cac(cc("#928374", "245", "8"), cc("#928374", "245", "8")),
	Primary:     cac(cc("#427b58", "65", "6"), cc("#8ec07c", "108", "14")),
	Secondary:   cac(cc("#076678", "24", "4"), cc("#83a598", "109", "12")),
	Accent:      cac(cc("#b57614", "136", "3"), cc("#fabd2f", "214", "11")),
	Success:     cac(cc("#79740e", "100", "2"), cc("#b8bb26", "142", "10")),
	Warning:     cac(cc("#b57614", "136", "3"), cc("#fabd2f", "214", "11")),
	Error:       cac(cc("#9d0006", "88", "1"), cc("#fb4934", "167", "9")),
	Border:      cac(cc("#928374", "245", "8"), cc("#665c54", "241", "8")),
	BorderFocus: cac(cc("#427b58", "65", "6"), cc("#8ec07c", "108", "14")),
}

var Nord = Palette{
	Name:        "nord",
	Base:        cac(cc("#eceff4", "255", "15"), cc("#2e3440", "236", "0")),
	Surface:     cac(cc("#e5e9f0", "254", "7"), cc("#3b4252", "237", "8")),
	Overlay:     cac(cc("#d8dee9", "253", "7"), cc("#434c5e", "238", "8")),
	Text:        cac(cc("#2e3440", "236", "0"), cc("#eceff4", "255", "15")),
	Subtext:     cac(cc("#3b4252", "237", "8"), cc("#e5e9f0", "254", "7")),
	Muted:       cac(cc("#4c566a", "240", "8"), cc("#4c566a", "240", "8")),
	Primary:     cac(cc("#5e81ac", "67", "4"), cc("#81a1c1", "110", "12")),
	Secondary:   cac(cc("#b48ead", "139", "5"), cc("#b48ead", "139", "13")),
	Accent:      cac(cc("#88c0d0", "110", "6"), cc("#88c0d0", "110", "14")),
	Success:     cac(cc("#a3be8c", "144", "2"), cc("#a3be8c", "144", "10")),
	Warning:     cac(cc("#ebcb8b", "222", "3"), cc("#ebcb8b", "222", "11")),
	Error:       cac(cc("#bf616a", "131", "1"), cc("#bf616a", "131", "9")),
	Border:      cac(cc("#4c566a", "240", "8"), cc("#4c566a", "240", "8")),
	BorderFocus: cac(cc("#5e81ac", "67", "4"), cc("#81a1c1", "110", "12")),
}
