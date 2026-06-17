package theme

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"github.com/charmbracelet/colorprofile"
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
var Palettes = []Palette{GhDash, Catppuccin, TomorrowNight, TokyoNight, Gruvbox, Monokai, Nord, RosePine, Solarized, Oxocarbon}

// PaletteByName returns the palette with the given name, or Catppuccin as default.
func PaletteByName(name string) *Palette {
	name = paletteAlias(name)
	for i := range Palettes {
		if Palettes[i].Name == name {
			return &Palettes[i]
		}
	}
	for i := range Palettes {
		if Palettes[i].Name == "rose pine" {
			return &Palettes[i]
		}
	}
	return &Palettes[0]
}

func paletteAlias(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "catppuccin mocha", "catppuccin-mocha":
		return "catppuccin"
	case "monokai remastered", "monokai-remastered":
		return "monokai"
	case "tomorrow night eighties", "tomorrow-night-eighties", "tomorrow-night":
		return "tomorrow night"
	case "rosé pine", "rose-pine", "rosepine", "rosé-pine":
		return "rose pine"
	case "solarized dark", "solarized-dark":
		return "solarized"
	default:
		return name
	}
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

func SetTerminalBackgroundDark(isDark bool) {
	compat.HasDarkBackground = isDark
}

func SetTerminalColorProfile(profile colorprofile.Profile) {
	compat.Profile = profile
}

func TerminalColorSupported() bool {
	switch compat.Profile {
	case colorprofile.ANSI, colorprofile.ANSI256, colorprofile.TrueColor:
		return true
	default:
		return false
	}
}

func ShouldRequestTrueColorCapability(profile colorprofile.Profile) bool {
	switch profile {
	case colorprofile.ANSI, colorprofile.ANSI256:
		return true
	default:
		return false
	}
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

func solid(trueColor, ansi256, ansi string) compat.CompleteAdaptiveColor {
	c := cc(trueColor, ansi256, ansi)
	return cac(c, c)
}

var Catppuccin = Palette{
	Name:        "catppuccin",
	Base:        solid("#1e1e2e", "235", "0"),
	Surface:     solid("#45475a", "239", "8"),
	Overlay:     solid("#585b70", "241", "8"),
	Text:        solid("#cdd6f4", "189", "15"),
	Subtext:     solid("#a6adc8", "146", "7"),
	Muted:       solid("#585b70", "241", "8"),
	Primary:     solid("#89b4fa", "111", "12"),
	Secondary:   solid("#f5c2e7", "218", "13"),
	Accent:      solid("#94e2d5", "116", "14"),
	Success:     solid("#a6e3a1", "151", "10"),
	Warning:     solid("#f9e2af", "223", "11"),
	Error:       solid("#f38ba8", "211", "9"),
	Border:      solid("#585b70", "241", "8"),
	BorderFocus: solid("#89b4fa", "111", "12"),
}

var TomorrowNight = Palette{
	Name:        "tomorrow night",
	Base:        solid("#2d2d2d", "236", "0"),
	Surface:     solid("#000000", "16", "0"),
	Overlay:     solid("#515151", "239", "8"),
	Text:        solid("#cccccc", "252", "7"),
	Subtext:     solid("#ffffff", "231", "15"),
	Muted:       solid("#595959", "240", "8"),
	Primary:     solid("#6699cc", "68", "12"),
	Secondary:   solid("#cc99cc", "176", "13"),
	Accent:      solid("#66cccc", "80", "14"),
	Success:     solid("#99cc99", "114", "10"),
	Warning:     solid("#ffcc66", "221", "11"),
	Error:       solid("#f2777a", "210", "9"),
	Border:      solid("#515151", "239", "8"),
	BorderFocus: solid("#6699cc", "68", "12"),
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
	Text:        cac(cc("#343b58", "237", "0"), cc("#d5dcff", "254", "15")),
	Subtext:     cac(cc("#4e5579", "240", "8"), cc("#b8c2f0", "252", "7")),
	Muted:       cac(cc("#6e7191", "243", "8"), cc("#6f7aae", "243", "8")),
	Primary:     cac(cc("#965027", "130", "3"), cc("#ff9e64", "215", "11")),
	Secondary:   cac(cc("#007197", "30", "6"), cc("#7dcfff", "117", "14")),
	Accent:      cac(cc("#965027", "130", "3"), cc("#ff9e64", "215", "11")),
	Success:     cac(cc("#587539", "64", "2"), cc("#9ece6a", "149", "10")),
	Warning:     cac(cc("#8c6c3e", "136", "3"), cc("#e0af68", "179", "11")),
	Error:       cac(cc("#c64343", "160", "1"), cc("#f7768e", "204", "9")),
	Border:      cac(cc("#9ca0b0", "246", "8"), cc("#414868", "238", "8")),
	BorderFocus: cac(cc("#965027", "130", "3"), cc("#ff9e64", "215", "11")),
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

var Monokai = Palette{
	Name:        "monokai",
	Base:        solid("#0c0c0c", "232", "0"),
	Surface:     solid("#1a1a1a", "234", "0"),
	Overlay:     solid("#343434", "236", "8"),
	Text:        solid("#d9d9d9", "253", "15"),
	Subtext:     solid("#c4c5b5", "250", "7"),
	Muted:       solid("#625e4c", "240", "8"),
	Primary:     solid("#9d65ff", "99", "12"),
	Secondary:   solid("#f4005f", "197", "13"),
	Accent:      solid("#58d1eb", "80", "14"),
	Success:     solid("#98e024", "112", "10"),
	Warning:     solid("#fd971f", "208", "11"),
	Error:       solid("#f4005f", "197", "9"),
	Border:      solid("#343434", "236", "8"),
	BorderFocus: solid("#9d65ff", "99", "12"),
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

// rose pine — dark base, rose primary (distinct from existing blue/orange/green/purple mains)
var RosePine = Palette{
	Name:        "rose pine",
	Base:        solid("#191724", "234", "0"),
	Surface:     solid("#1f1d2e", "235", "8"),
	Overlay:     solid("#26233a", "237", "8"),
	Text:        solid("#e0def4", "189", "15"),
	Subtext:     solid("#dcd9f0", "189", "15"),
	Muted:       solid("#6e6a86", "60", "8"),
	Primary:     solid("#ebbcba", "181", "13"),
	Secondary:   solid("#c4a7e7", "183", "13"),
	Accent:      solid("#9ccfd8", "152", "14"),
	Success:     solid("#31748f", "66", "6"),
	Warning:     solid("#f6c177", "222", "11"),
	Error:       solid("#eb6f92", "204", "9"),
	Border:      solid("#26233a", "237", "8"),
	BorderFocus: solid("#ebbcba", "181", "13"),
}

// solarized dark — distinctive dark-teal base, cyan primary
var Solarized = Palette{
	Name:        "solarized",
	Base:        solid("#002b36", "234", "0"),
	Surface:     solid("#073642", "235", "8"),
	Overlay:     solid("#586e75", "240", "8"),
	Text:        solid("#93a1a1", "245", "7"),
	Subtext:     solid("#839496", "244", "7"),
	Muted:       solid("#586e75", "240", "8"),
	Primary:     solid("#2aa198", "37", "6"),
	Secondary:   solid("#6c71c4", "61", "5"),
	Accent:      solid("#268bd2", "33", "4"),
	Success:     solid("#859900", "64", "2"),
	Warning:     solid("#b58900", "136", "3"),
	Error:       solid("#dc322f", "160", "1"),
	Border:      solid("#073642", "235", "8"),
	BorderFocus: solid("#2aa198", "37", "6"),
}

// oxocarbon — ibm carbon dark, magenta primary
var Oxocarbon = Palette{
	Name:        "oxocarbon",
	Base:        solid("#161616", "233", "0"),
	Surface:     solid("#262626", "235", "8"),
	Overlay:     solid("#393939", "237", "8"),
	Text:        solid("#f2f4f8", "255", "15"),
	Subtext:     solid("#dde1e6", "253", "7"),
	Muted:       solid("#525252", "240", "8"),
	Primary:     solid("#ee5396", "205", "13"),
	Secondary:   solid("#be95ff", "141", "13"),
	Accent:      solid("#3ddbd9", "80", "14"),
	Success:     solid("#42be65", "78", "10"),
	Warning:     solid("#ff832b", "208", "11"),
	Error:       solid("#fa4d56", "203", "9"),
	Border:      solid("#393939", "237", "8"),
	BorderFocus: solid("#ee5396", "205", "13"),
}
