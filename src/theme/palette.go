package theme

import "github.com/charmbracelet/lipgloss"

// Palette defines a complete set of colors for theming.
// uses lipgloss.CompleteAdaptiveColor for light/dark terminal support.
type Palette struct {
	Name string

	// primary surfaces
	Base    lipgloss.CompleteAdaptiveColor
	Surface lipgloss.CompleteAdaptiveColor
	Overlay lipgloss.CompleteAdaptiveColor

	// text
	Text    lipgloss.CompleteAdaptiveColor
	Subtext lipgloss.CompleteAdaptiveColor
	Muted   lipgloss.CompleteAdaptiveColor

	// accents
	Primary   lipgloss.CompleteAdaptiveColor
	Secondary lipgloss.CompleteAdaptiveColor
	Accent    lipgloss.CompleteAdaptiveColor

	// semantic
	Success lipgloss.CompleteAdaptiveColor
	Warning lipgloss.CompleteAdaptiveColor
	Error   lipgloss.CompleteAdaptiveColor

	// borders
	Border      lipgloss.CompleteAdaptiveColor
	BorderFocus lipgloss.CompleteAdaptiveColor
}

// built-in palettes
var Palettes = []Palette{Catppuccin, TokyoNight, Gruvbox, Nord}

// PaletteByName returns the palette with the given name, or Catppuccin as default.
func PaletteByName(name string) *Palette {
	for i := range Palettes {
		if Palettes[i].Name == name {
			return &Palettes[i]
		}
	}
	return &Palettes[0]
}

// PaletteNames returns all available palette names.
func PaletteNames() []string {
	names := make([]string, len(Palettes))
	for i := range Palettes {
		names[i] = Palettes[i].Name
	}
	return names
}

var Catppuccin = Palette{
	Name:        "catppuccin",
	Base:        lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#eff1f5", ANSI256: "255", ANSI: "15"}, Dark: lipgloss.CompleteColor{TrueColor: "#1e1e2e", ANSI256: "234", ANSI: "0"}},
	Surface:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#ccd0da", ANSI256: "252", ANSI: "7"}, Dark: lipgloss.CompleteColor{TrueColor: "#313244", ANSI256: "236", ANSI: "8"}},
	Overlay:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#bcc0cc", ANSI256: "250", ANSI: "7"}, Dark: lipgloss.CompleteColor{TrueColor: "#45475a", ANSI256: "238", ANSI: "8"}},
	Text:        lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#4c4f69", ANSI256: "239", ANSI: "0"}, Dark: lipgloss.CompleteColor{TrueColor: "#cdd6f4", ANSI256: "254", ANSI: "15"}},
	Subtext:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#6c6f85", ANSI256: "242", ANSI: "8"}, Dark: lipgloss.CompleteColor{TrueColor: "#bac2de", ANSI256: "252", ANSI: "7"}},
	Muted:       lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#9ca0b0", ANSI256: "246", ANSI: "8"}, Dark: lipgloss.CompleteColor{TrueColor: "#6c7086", ANSI256: "242", ANSI: "8"}},
	Primary:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#8839ef", ANSI256: "129", ANSI: "5"}, Dark: lipgloss.CompleteColor{TrueColor: "#cba6f7", ANSI256: "183", ANSI: "13"}},
	Secondary:   lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#1e66f5", ANSI256: "33", ANSI: "4"}, Dark: lipgloss.CompleteColor{TrueColor: "#89b4fa", ANSI256: "111", ANSI: "12"}},
	Accent:      lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#179299", ANSI256: "30", ANSI: "6"}, Dark: lipgloss.CompleteColor{TrueColor: "#94e2d5", ANSI256: "115", ANSI: "14"}},
	Success:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#40a02b", ANSI256: "34", ANSI: "2"}, Dark: lipgloss.CompleteColor{TrueColor: "#a6e3a1", ANSI256: "114", ANSI: "10"}},
	Warning:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#df8e1d", ANSI256: "172", ANSI: "3"}, Dark: lipgloss.CompleteColor{TrueColor: "#f9e2af", ANSI256: "223", ANSI: "11"}},
	Error:       lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#d20f39", ANSI256: "160", ANSI: "1"}, Dark: lipgloss.CompleteColor{TrueColor: "#f38ba8", ANSI256: "211", ANSI: "9"}},
	Border:      lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#9ca0b0", ANSI256: "246", ANSI: "8"}, Dark: lipgloss.CompleteColor{TrueColor: "#585b70", ANSI256: "240", ANSI: "8"}},
	BorderFocus: lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#8839ef", ANSI256: "129", ANSI: "5"}, Dark: lipgloss.CompleteColor{TrueColor: "#cba6f7", ANSI256: "183", ANSI: "13"}},
}

var TokyoNight = Palette{
	Name:        "tokyonight",
	Base:        lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#d5d6db", ANSI256: "253", ANSI: "15"}, Dark: lipgloss.CompleteColor{TrueColor: "#1a1b26", ANSI256: "234", ANSI: "0"}},
	Surface:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#c4c5cb", ANSI256: "251", ANSI: "7"}, Dark: lipgloss.CompleteColor{TrueColor: "#24283b", ANSI256: "235", ANSI: "8"}},
	Overlay:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#b4b5bb", ANSI256: "249", ANSI: "7"}, Dark: lipgloss.CompleteColor{TrueColor: "#414868", ANSI256: "238", ANSI: "8"}},
	Text:        lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#343b58", ANSI256: "237", ANSI: "0"}, Dark: lipgloss.CompleteColor{TrueColor: "#c0caf5", ANSI256: "253", ANSI: "15"}},
	Subtext:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#4e5579", ANSI256: "240", ANSI: "8"}, Dark: lipgloss.CompleteColor{TrueColor: "#a9b1d6", ANSI256: "249", ANSI: "7"}},
	Muted:       lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#6e7191", ANSI256: "243", ANSI: "8"}, Dark: lipgloss.CompleteColor{TrueColor: "#565f89", ANSI256: "242", ANSI: "8"}},
	Primary:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#7830f0", ANSI256: "93", ANSI: "5"}, Dark: lipgloss.CompleteColor{TrueColor: "#bb9af7", ANSI256: "141", ANSI: "13"}},
	Secondary:   lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#2e7de9", ANSI256: "33", ANSI: "4"}, Dark: lipgloss.CompleteColor{TrueColor: "#7aa2f7", ANSI256: "111", ANSI: "12"}},
	Accent:      lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#007197", ANSI256: "30", ANSI: "6"}, Dark: lipgloss.CompleteColor{TrueColor: "#7dcfff", ANSI256: "117", ANSI: "14"}},
	Success:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#587539", ANSI256: "64", ANSI: "2"}, Dark: lipgloss.CompleteColor{TrueColor: "#9ece6a", ANSI256: "149", ANSI: "10"}},
	Warning:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#8c6c3e", ANSI256: "136", ANSI: "3"}, Dark: lipgloss.CompleteColor{TrueColor: "#e0af68", ANSI256: "179", ANSI: "11"}},
	Error:       lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#c64343", ANSI256: "160", ANSI: "1"}, Dark: lipgloss.CompleteColor{TrueColor: "#f7768e", ANSI256: "204", ANSI: "9"}},
	Border:      lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#9ca0b0", ANSI256: "246", ANSI: "8"}, Dark: lipgloss.CompleteColor{TrueColor: "#414868", ANSI256: "238", ANSI: "8"}},
	BorderFocus: lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#7830f0", ANSI256: "93", ANSI: "5"}, Dark: lipgloss.CompleteColor{TrueColor: "#bb9af7", ANSI256: "141", ANSI: "13"}},
}

var Gruvbox = Palette{
	Name:        "gruvbox",
	Base:        lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#fbf1c7", ANSI256: "230", ANSI: "15"}, Dark: lipgloss.CompleteColor{TrueColor: "#282828", ANSI256: "235", ANSI: "0"}},
	Surface:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#ebdbb2", ANSI256: "223", ANSI: "7"}, Dark: lipgloss.CompleteColor{TrueColor: "#3c3836", ANSI256: "237", ANSI: "8"}},
	Overlay:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#d5c4a1", ANSI256: "187", ANSI: "7"}, Dark: lipgloss.CompleteColor{TrueColor: "#504945", ANSI256: "239", ANSI: "8"}},
	Text:        lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#3c3836", ANSI256: "237", ANSI: "0"}, Dark: lipgloss.CompleteColor{TrueColor: "#ebdbb2", ANSI256: "223", ANSI: "15"}},
	Subtext:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#504945", ANSI256: "239", ANSI: "8"}, Dark: lipgloss.CompleteColor{TrueColor: "#d5c4a1", ANSI256: "187", ANSI: "7"}},
	Muted:       lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#928374", ANSI256: "245", ANSI: "8"}, Dark: lipgloss.CompleteColor{TrueColor: "#928374", ANSI256: "245", ANSI: "8"}},
	Primary:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#9d0006", ANSI256: "88", ANSI: "1"}, Dark: lipgloss.CompleteColor{TrueColor: "#fb4934", ANSI256: "167", ANSI: "9"}},
	Secondary:   lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#076678", ANSI256: "24", ANSI: "4"}, Dark: lipgloss.CompleteColor{TrueColor: "#83a598", ANSI256: "109", ANSI: "12"}},
	Accent:      lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#427b58", ANSI256: "65", ANSI: "6"}, Dark: lipgloss.CompleteColor{TrueColor: "#8ec07c", ANSI256: "108", ANSI: "14"}},
	Success:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#79740e", ANSI256: "100", ANSI: "2"}, Dark: lipgloss.CompleteColor{TrueColor: "#b8bb26", ANSI256: "142", ANSI: "10"}},
	Warning:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#b57614", ANSI256: "136", ANSI: "3"}, Dark: lipgloss.CompleteColor{TrueColor: "#fabd2f", ANSI256: "214", ANSI: "11"}},
	Error:       lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#9d0006", ANSI256: "88", ANSI: "1"}, Dark: lipgloss.CompleteColor{TrueColor: "#fb4934", ANSI256: "167", ANSI: "9"}},
	Border:      lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#928374", ANSI256: "245", ANSI: "8"}, Dark: lipgloss.CompleteColor{TrueColor: "#665c54", ANSI256: "241", ANSI: "8"}},
	BorderFocus: lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#9d0006", ANSI256: "88", ANSI: "1"}, Dark: lipgloss.CompleteColor{TrueColor: "#fb4934", ANSI256: "167", ANSI: "9"}},
}

var Nord = Palette{
	Name:        "nord",
	Base:        lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#eceff4", ANSI256: "255", ANSI: "15"}, Dark: lipgloss.CompleteColor{TrueColor: "#2e3440", ANSI256: "236", ANSI: "0"}},
	Surface:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#e5e9f0", ANSI256: "254", ANSI: "7"}, Dark: lipgloss.CompleteColor{TrueColor: "#3b4252", ANSI256: "237", ANSI: "8"}},
	Overlay:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#d8dee9", ANSI256: "253", ANSI: "7"}, Dark: lipgloss.CompleteColor{TrueColor: "#434c5e", ANSI256: "238", ANSI: "8"}},
	Text:        lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#2e3440", ANSI256: "236", ANSI: "0"}, Dark: lipgloss.CompleteColor{TrueColor: "#eceff4", ANSI256: "255", ANSI: "15"}},
	Subtext:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#3b4252", ANSI256: "237", ANSI: "8"}, Dark: lipgloss.CompleteColor{TrueColor: "#e5e9f0", ANSI256: "254", ANSI: "7"}},
	Muted:       lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#4c566a", ANSI256: "240", ANSI: "8"}, Dark: lipgloss.CompleteColor{TrueColor: "#4c566a", ANSI256: "240", ANSI: "8"}},
	Primary:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#5e81ac", ANSI256: "67", ANSI: "4"}, Dark: lipgloss.CompleteColor{TrueColor: "#81a1c1", ANSI256: "110", ANSI: "12"}},
	Secondary:   lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#b48ead", ANSI256: "139", ANSI: "5"}, Dark: lipgloss.CompleteColor{TrueColor: "#b48ead", ANSI256: "139", ANSI: "13"}},
	Accent:      lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#88c0d0", ANSI256: "110", ANSI: "6"}, Dark: lipgloss.CompleteColor{TrueColor: "#88c0d0", ANSI256: "110", ANSI: "14"}},
	Success:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#a3be8c", ANSI256: "144", ANSI: "2"}, Dark: lipgloss.CompleteColor{TrueColor: "#a3be8c", ANSI256: "144", ANSI: "10"}},
	Warning:     lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#ebcb8b", ANSI256: "222", ANSI: "3"}, Dark: lipgloss.CompleteColor{TrueColor: "#ebcb8b", ANSI256: "222", ANSI: "11"}},
	Error:       lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#bf616a", ANSI256: "131", ANSI: "1"}, Dark: lipgloss.CompleteColor{TrueColor: "#bf616a", ANSI256: "131", ANSI: "9"}},
	Border:      lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#4c566a", ANSI256: "240", ANSI: "8"}, Dark: lipgloss.CompleteColor{TrueColor: "#4c566a", ANSI256: "240", ANSI: "8"}},
	BorderFocus: lipgloss.CompleteAdaptiveColor{Light: lipgloss.CompleteColor{TrueColor: "#5e81ac", ANSI256: "67", ANSI: "4"}, Dark: lipgloss.CompleteColor{TrueColor: "#81a1c1", ANSI256: "110", ANSI: "12"}},
}
