package main

import (
	"context"

	"github.com/emin/konfigurator/pkg"
)

type ghosttyGenerator struct{}

func (g *ghosttyGenerator) Generate(_ context.Context) (*pkg.Schema, error) {
	return &pkg.Schema{
		App:     "ghostty",
		Format:  "ghostty",
		DocsURL: "https://ghostty.org/docs/config",
		Sections: []pkg.Section{
			appearance(),
			window(),
			shell(),
		},
	}, nil
}

func appearance() pkg.Section {
	return pkg.Section{
		Name: "Appearance",
		Fields: []pkg.Field{
			{
				Key:         "font-family",
				Label:       "Font Family",
				Type:        "string",
				Description: "Primary font family",
				Example:     `font-family = "JetBrains Mono"`,
				Hint:        "must be a font installed on the system",
				DocURL:      "https://ghostty.org/docs/config/reference#font-family",
				Since:       "1.0.0",
			},
			{
				Key:         "font-size",
				Label:       "Font Size",
				Type:        "number",
				Default:     "13",
				Description: "Font size in points",
				Min:         floatPtr(6),
				Max:         floatPtr(72),
				Example:     "font-size = 14",
				DocURL:      "https://ghostty.org/docs/config/reference#font-size",
				Since:       "1.0.0",
			},
			{
				Key:         "background",
				Label:       "Background",
				Type:        "color",
				Default:     "282828",
				Description: "Background color (hex without #)",
			},
			{
				Key:         "foreground",
				Label:       "Foreground",
				Type:        "color",
				Default:     "ebdbb2",
				Description: "Foreground text color (hex without #)",
			},
			{
				Key:         "cursor-color",
				Label:       "Cursor Color",
				Type:        "color",
				Description: "Cursor color",
			},
			{
				Key:         "cursor-style",
				Label:       "Cursor Style",
				Type:        "enum",
				Default:     "block",
				Description: "Cursor shape",
				Options:     []string{"block", "bar", "underline"},
				Example:     "cursor-style = bar",
				Hint:        "bar works well with blinking enabled",
				DocURL:      "https://ghostty.org/docs/config/reference#cursor-style",
				Since:       "1.0.0",
			},
		},
	}
}

func window() pkg.Section {
	return pkg.Section{
		Name: "Window",
		Fields: []pkg.Field{
			{
				Key:         "window-decoration",
				Label:       "Window Decoration",
				Type:        "bool",
				Default:     "true",
				Description: "Show window decorations",
			},
			{
				Key:         "window-padding-x",
				Label:       "Padding X",
				Type:        "number",
				Default:     "0",
				Description: "Horizontal window padding in pixels",
				Min:         floatPtr(0),
				Max:         floatPtr(100),
			},
			{
				Key:         "window-padding-y",
				Label:       "Padding Y",
				Type:        "number",
				Default:     "0",
				Description: "Vertical window padding in pixels",
				Min:         floatPtr(0),
				Max:         floatPtr(100),
			},
		},
	}
}

func shell() pkg.Section {
	return pkg.Section{
		Name: "Shell",
		Fields: []pkg.Field{
			{
				Key:         "shell-integration",
				Label:       "Shell Integration",
				Type:        "enum",
				Default:     "detect",
				Description: "Shell integration mode",
				Options:     []string{"none", "detect", "bash", "zsh", "fish"},
			},
			{
				Key:         "shell-integration-features",
				Label:       "Integration Features",
				Type:        "string",
				Description: "Comma-separated shell integration features",
			},
		},
	}
}

func floatPtr(f float64) *float64 { return &f }
