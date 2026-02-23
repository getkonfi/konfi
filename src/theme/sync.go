package theme

// AppSync defines how a konfable maps palette colors to its config keys.
// data only — orchestration lives in setup/ or main.
type AppSync struct {
	AppName string
	Enabled bool

	// maps konfable config keys to palette color accessors.
	// e.g. {"background": "Base", "foreground": "Text"}
	Mappings map[string]string

	// FormatColor converts a hex color (#rrggbb) to the format the app expects.
	// e.g. ghostty wants "rrggbb" (no #), alacritty wants "0xrrggbb".
	FormatColor func(hex string) string
}
