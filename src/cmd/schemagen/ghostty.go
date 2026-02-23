package main

import (
	"context"

	"github.com/emin/konfigurator/pkg"
)

type ghosttyGenerator struct{}

func (g *ghosttyGenerator) Generate(_ context.Context) (*pkg.Schema, error) {
	return &pkg.Schema{
		App:           "ghostty",
		Format:        "ghostty",
		SchemaVersion: "1.1.0",
		DocsURL:       "https://ghostty.org/docs/config",
		Hints: []string{
			"gpu-accelerated terminal with zero-config defaults",
			"supports ligatures and font features via font-feature",
			"use keybind to customize shortcuts",
		},
		Sections: []pkg.Section{
			ghosttyFontFamily(),
			ghosttyFontStyle(),
			ghosttyFontAdvanced(),
			ghosttyColorTheme(),
			ghosttyColorSelection(),
			ghosttyColorAppearance(),
			ghosttyCursor(),
			ghosttyWindowChrome(),
			ghosttyWindowPadding(),
			ghosttyWindowState(),
			ghosttyCellAdjust(),
			ghosttyInput(),
			ghosttyShell(),
			ghosttyClipboard(),
			ghosttyAdvanced(),
		},
	}, nil
}

func ghosttyFontFamily() pkg.Section {
	return pkg.Section{
		Name: "Font Family",
		Fields: []pkg.Field{
			{
				Key:         "font-family",
				Label:       "Font Family",
				Type:        "string",
				Description: "primary font family",
				Example:     `font-family = JetBrains Mono`,
				Hint:        "must be a font installed on the system",
				DocURL:      "https://ghostty.org/docs/config/reference#font-family",
				Since:       "1.0.0",
			},
			{
				Key:         "font-family-bold",
				Label:       "Bold Font",
				Type:        "string",
				Description: "font family for bold text",
				Since:       "1.0.0",
			},
			{
				Key:         "font-family-italic",
				Label:       "Italic Font",
				Type:        "string",
				Description: "font family for italic text",
				Since:       "1.0.0",
			},
			{
				Key:         "font-family-bold-italic",
				Label:       "Bold Italic Font",
				Type:        "string",
				Description: "font family for bold italic text",
				Since:       "1.0.0",
			},
		},
	}
}

func ghosttyFontStyle() pkg.Section {
	return pkg.Section{
		Name: "Font Style",
		Fields: []pkg.Field{
			{
				Key:         "font-size",
				Label:       "Font Size",
				Type:        "number",
				Default:     "13",
				Description: "font size in points",
				Min:         floatPtr(6),
				Max:         floatPtr(72),
				DocURL:      "https://ghostty.org/docs/config/reference#font-size",
				Since:       "1.0.0",
			},
			{
				Key:         "font-style",
				Label:       "Font Style",
				Type:        "string",
				Description: "default font style (e.g., regular, medium, light)",
				Since:       "1.0.0",
			},
			{
				Key:         "font-style-bold",
				Label:       "Bold Style",
				Type:        "string",
				Description: "font style to use for bold",
				Since:       "1.0.0",
			},
			{
				Key:         "font-style-italic",
				Label:       "Italic Style",
				Type:        "string",
				Description: "font style to use for italic",
				Since:       "1.0.0",
			},
		},
	}
}

func ghosttyFontAdvanced() pkg.Section {
	return pkg.Section{
		Name: "Font Advanced",
		Fields: []pkg.Field{
			{
				Key:         "font-feature",
				Label:       "Font Feature",
				Type:        "list",
				Description: "OpenType font features to enable (e.g., ss01, calt)",
				Hint:        "repeated key — one feature per line",
				Since:       "1.0.0",
			},
			{
				Key:         "font-variation",
				Label:       "Font Variation",
				Type:        "list",
				Description: "font variation axis settings (e.g., wght=400)",
				Since:       "1.0.0",
			},
			{
				Key:         "font-thicken",
				Label:       "Font Thicken",
				Type:        "bool",
				Default:     "false",
				Description: "thicken font glyphs for better visibility on HiDPI",
				Since:       "1.0.0",
			},
		},
	}
}

func ghosttyColorTheme() pkg.Section {
	gruvboxBg := []string{"282828", "1d2021", "32302f", "3c3836", "504945", "665c54", "1a1b26", "24283b", "011628", "002b36"}
	gruvboxFg := []string{"ebdbb2", "fbf1c7", "d5c4a1", "bdae93", "a9b1d6", "c0caf5", "a89984", "839496", "93a1a1", "fdf6e3"}

	return pkg.Section{
		Name: "Color Theme",
		Fields: []pkg.Field{
			{
				Key:         "theme",
				Label:       "Theme",
				Type:        "string",
				Description: "color theme name (overrides individual colors)",
				Hint:        "use 'ghostty +list-themes' to see available themes",
			},
			{
				Key:         "background",
				Label:       "Background",
				Type:        "color",
				Default:     "282828",
				Description: "background color (hex without #)",
				Palette:     gruvboxBg,
			},
			{
				Key:         "foreground",
				Label:       "Foreground",
				Type:        "color",
				Default:     "ebdbb2",
				Description: "foreground text color (hex without #)",
				Palette:     gruvboxFg,
			},
		},
	}
}

func ghosttyColorSelection() pkg.Section {
	gruvboxBg := []string{"282828", "1d2021", "32302f", "3c3836", "504945", "665c54", "1a1b26", "24283b", "011628", "002b36"}
	gruvboxFg := []string{"ebdbb2", "fbf1c7", "d5c4a1", "bdae93", "a9b1d6", "c0caf5", "a89984", "839496", "93a1a1", "fdf6e3"}
	gruvboxAccent := []string{"ebdbb2", "cc241d", "d65d0e", "d79921", "98971a", "689d6a", "458588", "b16286"}

	return pkg.Section{
		Name: "Color Selection",
		Fields: []pkg.Field{
			{
				Key:         "selection-foreground",
				Label:       "Selection FG",
				Type:        "color",
				Description: "text color in selected regions",
				Palette:     gruvboxFg,
			},
			{
				Key:         "selection-background",
				Label:       "Selection BG",
				Type:        "color",
				Description: "background color in selected regions",
				Palette:     gruvboxBg,
			},
			{
				Key:         "cursor-color",
				Label:       "Cursor Color",
				Type:        "color",
				Description: "cursor color",
				Palette:     gruvboxAccent,
			},
			{
				Key:         "cursor-text",
				Label:       "Cursor Text",
				Type:        "color",
				Description: "text color under the cursor",
				Palette:     gruvboxBg,
			},
			{
				Key:         "unfocused-split-fill",
				Label:       "Unfocused Fill",
				Type:        "color",
				Description: "fill color for unfocused splits",
			},
		},
	}
}

func ghosttyColorAppearance() pkg.Section {
	return pkg.Section{
		Name: "Color Appearance",
		Fields: []pkg.Field{
			{
				Key:         "background-opacity",
				Label:       "BG Opacity",
				Type:        "number",
				Default:     "1.0",
				Description: "background opacity (0.0–1.0)",
				Min:         floatPtr(0),
				Max:         floatPtr(1),
			},
			{
				Key:         "minimum-contrast",
				Label:       "Min Contrast",
				Type:        "number",
				Default:     "1",
				Description: "minimum contrast ratio (1–21)",
				Min:         floatPtr(1),
				Max:         floatPtr(21),
			},
		},
	}
}

func ghosttyCursor() pkg.Section {
	return pkg.Section{
		Name: "Cursor",
		Fields: []pkg.Field{
			{
				Key:         "cursor-style",
				Label:       "Cursor Style",
				Type:        "enum",
				Default:     "block",
				Description: "cursor shape",
				Options:     []string{"block", "bar", "underline"},
				DocURL:      "https://ghostty.org/docs/config/reference#cursor-style",
				Since:       "1.0.0",
			},
			{
				Key:         "cursor-style-blink",
				Label:       "Cursor Blink",
				Type:        "bool",
				Default:     "true",
				Description: "whether the cursor blinks",
			},
			{
				Key:         "cursor-opacity",
				Label:       "Cursor Opacity",
				Type:        "number",
				Default:     "1.0",
				Description: "cursor opacity (0.0–1.0)",
				Min:         floatPtr(0),
				Max:         floatPtr(1),
			},
			{
				Key:         "cursor-click-to-move",
				Label:       "Click to Move",
				Type:        "bool",
				Default:     "false",
				Description: "click to reposition cursor in supporting apps",
			},
			{
				Key:         "mouse-hide-while-typing",
				Label:       "Hide Mouse Typing",
				Type:        "bool",
				Default:     "false",
				Description: "hide the mouse cursor while typing",
			},
		},
	}
}

func ghosttyWindowChrome() pkg.Section {
	return pkg.Section{
		Name: "Window Chrome",
		Fields: []pkg.Field{
			{
				Key:         "window-decoration",
				Label:       "Window Decoration",
				Type:        "bool",
				Default:     "true",
				Description: "show window decorations (title bar, etc.)",
			},
			{
				Key:         "window-title-font-family",
				Label:       "Title Font",
				Type:        "string",
				Description: "font family for window title bar (macOS)",
			},
			{
				Key:         "window-inherit-font-size",
				Label:       "Inherit Font Size",
				Type:        "bool",
				Default:     "true",
				Description: "new windows inherit font size from focused window",
			},
			{
				Key:         "title",
				Label:       "Title",
				Type:        "string",
				Description: "fixed window title (overrides dynamic title)",
			},
		},
	}
}

func ghosttyWindowPadding() pkg.Section {
	return pkg.Section{
		Name: "Window Padding",
		Fields: []pkg.Field{
			{
				Key:         "window-padding-x",
				Label:       "Padding X",
				Type:        "number",
				Default:     "0",
				Description: "horizontal window padding in pixels",
				Min:         floatPtr(0),
				Max:         floatPtr(200),
			},
			{
				Key:         "window-padding-y",
				Label:       "Padding Y",
				Type:        "number",
				Default:     "0",
				Description: "vertical window padding in pixels",
				Min:         floatPtr(0),
				Max:         floatPtr(200),
			},
			{
				Key:         "window-padding-balance",
				Label:       "Pad Balance",
				Type:        "bool",
				Default:     "false",
				Description: "automatically balance padding on both sides",
			},
			{
				Key:         "window-padding-color",
				Label:       "Pad Color",
				Type:        "enum",
				Default:     "background",
				Description: "color source for padding area",
				Options:     []string{"background", "extend", "extend-always"},
			},
		},
	}
}

func ghosttyWindowState() pkg.Section {
	return pkg.Section{
		Name: "Window State",
		Fields: []pkg.Field{
			{
				Key:         "window-save-state",
				Label:       "Save State",
				Type:        "enum",
				Default:     "default",
				Description: "save/restore window position and size",
				Options:     []string{"default", "never", "always"},
			},
			{
				Key:         "fullscreen",
				Label:       "Fullscreen",
				Type:        "bool",
				Default:     "false",
				Description: "start in fullscreen mode",
			},
			{
				Key:         "maximize",
				Label:       "Maximize",
				Type:        "bool",
				Default:     "false",
				Description: "start in maximized mode",
			},
		},
	}
}

func ghosttyCellAdjust() pkg.Section {
	return pkg.Section{
		Name: "Cell Adjustment",
		Fields: []pkg.Field{
			{
				Key:         "adjust-cell-width",
				Label:       "Cell Width",
				Type:        "string",
				Description: "cell width adjustment (e.g., +1, -1, 10%)",
				Hint:        "can be absolute (+1) or percentage (10%)",
			},
			{
				Key:         "adjust-cell-height",
				Label:       "Cell Height",
				Type:        "string",
				Description: "cell height adjustment (e.g., +1, -1, 10%)",
			},
			{
				Key:         "adjust-font-baseline",
				Label:       "Font Baseline",
				Type:        "string",
				Description: "font baseline adjustment (e.g., +1, -1)",
			},
			{
				Key:         "adjust-underline-position",
				Label:       "Underline Pos",
				Type:        "string",
				Description: "underline position adjustment",
			},
			{
				Key:         "adjust-underline-thickness",
				Label:       "Underline Thick",
				Type:        "string",
				Description: "underline thickness adjustment",
			},
			{
				Key:         "adjust-cursor-thickness",
				Label:       "Cursor Thick",
				Type:        "string",
				Description: "cursor bar/underline thickness adjustment",
			},
		},
	}
}

func ghosttyInput() pkg.Section {
	return pkg.Section{
		Name: "Input",
		Fields: []pkg.Field{
			{
				Key:         "keybind",
				Label:       "Key Bindings",
				Type:        "list",
				Description: "custom key bindings (repeated key)",
				Hint:        "format: trigger=action (e.g., ctrl+c=copy_to_clipboard)",
			},
			{
				Key:         "mouse-scroll-multiplier",
				Label:       "Scroll Multiplier",
				Type:        "number",
				Default:     "1",
				Description: "mouse scroll speed multiplier",
				Min:         floatPtr(0),
				Max:         floatPtr(50),
			},
			{
				Key:         "focus-follows-mouse",
				Label:       "Focus Follows Mouse",
				Type:        "bool",
				Default:     "false",
				Description: "split focus follows mouse pointer",
			},
			{
				Key:         "macos-option-as-alt",
				Label:       "Option as Alt",
				Type:        "enum",
				Default:     "false",
				Description: "treat macOS Option key as Alt",
				Options:     []string{"false", "true", "left", "right"},
			},
			{
				Key:         "link-url",
				Label:       "Clickable URLs",
				Type:        "bool",
				Default:     "true",
				Description: "detect and make URLs clickable",
			},
			{
				Key:         "mouse-shift-capture",
				Label:       "Shift+Click",
				Type:        "enum",
				Default:     "false",
				Description: "shift+click capture behavior",
				Options:     []string{"false", "true", "always"},
			},
		},
	}
}

func ghosttyShell() pkg.Section {
	return pkg.Section{
		Name: "Shell",
		Fields: []pkg.Field{
			{
				Key:         "command",
				Label:       "Shell Command",
				Type:        "string",
				Description: "command to run instead of default shell",
				Hint:        "leave empty for login shell",
			},
			{
				Key:         "shell-integration",
				Label:       "Shell Integration",
				Type:        "enum",
				Default:     "detect",
				Description: "shell integration mode",
				Options:     []string{"none", "detect", "bash", "zsh", "fish", "elvish"},
			},
			{
				Key:         "shell-integration-features",
				Label:       "Integration Features",
				Type:        "multi",
				Description: "shell integration features to enable",
				Options:     []string{"cursor", "sudo", "title", "no-cursor", "no-sudo", "no-title"},
			},
			{
				Key:         "scrollback-limit",
				Label:       "Scrollback Limit",
				Type:        "number",
				Default:     "10000",
				Description: "max scrollback lines (0 = unlimited)",
				Min:         floatPtr(0),
				Max:         floatPtr(1000000),
			},
			{
				Key:         "wait-after-command",
				Label:       "Wait After Exit",
				Type:        "bool",
				Default:     "false",
				Description: "keep surface open after command exits",
			},
			{
				Key:         "abnormal-command-exit-runtime",
				Label:       "Abnormal Exit Time",
				Type:        "number",
				Default:     "250",
				Description: "ms threshold for abnormal exit detection",
				Min:         floatPtr(0),
				Max:         floatPtr(60000),
			},
		},
	}
}

func ghosttyClipboard() pkg.Section {
	return pkg.Section{
		Name: "Clipboard",
		Fields: []pkg.Field{
			{
				Key:         "copy-on-select",
				Label:       "Copy on Select",
				Type:        "enum",
				Default:     "false",
				Description: "copy to clipboard on text selection",
				Options:     []string{"false", "true", "clipboard", "primary"},
			},
			{
				Key:         "clipboard-read",
				Label:       "Clipboard Read",
				Type:        "enum",
				Default:     "ask",
				Description: "allow programs to read clipboard",
				Options:     []string{"allow", "deny", "ask"},
			},
			{
				Key:         "clipboard-write",
				Label:       "Clipboard Write",
				Type:        "enum",
				Default:     "allow",
				Description: "allow programs to write clipboard",
				Options:     []string{"allow", "deny", "ask"},
			},
			{
				Key:         "clipboard-trim-trailing-spaces",
				Label:       "Trim Trailing",
				Type:        "bool",
				Default:     "true",
				Description: "trim trailing spaces when copying",
			},
			{
				Key:         "clipboard-paste-protection",
				Label:       "Paste Protection",
				Type:        "bool",
				Default:     "true",
				Description: "warn before pasting potentially unsafe text",
			},
		},
	}
}

func ghosttyAdvanced() pkg.Section {
	return pkg.Section{
		Name: "Advanced",
		Fields: []pkg.Field{
			{
				Key:         "config-file",
				Label:       "Extra Config",
				Type:        "list",
				Description: "additional config files to load",
			},
			{
				Key:         "confirm-close-surface",
				Label:       "Confirm Close",
				Type:        "bool",
				Default:     "true",
				Description: "confirm before closing a terminal surface",
			},
			{
				Key:         "image-storage-limit",
				Label:       "Image Limit",
				Type:        "number",
				Default:     "320000000",
				Description: "max bytes for in-memory image storage",
				Min:         floatPtr(0),
			},
			{
				Key:         "class",
				Label:       "Window Class",
				Type:        "string",
				Description: "X11/Wayland window class override",
			},
			{
				Key:         "custom-shader",
				Label:       "Custom Shader",
				Type:        "list",
				Description: "GLSL shader files for post-processing",
				Hint:        "path to .glsl files",
			},
			{
				Key:         "bold-is-bright",
				Label:       "Bold is Bright",
				Type:        "bool",
				Default:     "false",
				Description: "render bold text as bright color variant",
			},
			{
				Key:         "grapheme-width-method",
				Label:       "Grapheme Width",
				Type:        "enum",
				Default:     "unicode",
				Description: "method for computing grapheme width",
				Options:     []string{"unicode", "wcswidth", "legacy"},
			},
		},
	}
}

func floatPtr(f float64) *float64 { return &f }
