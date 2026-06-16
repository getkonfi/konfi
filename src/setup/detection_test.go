package setup

import (
	"testing"

	"github.com/getkonfi/konfi/konfables"
)

func TestAllKonfablesHaveLogos(t *testing.T) {
	for _, entry := range AllKonfablesWithInfo() {
		name := entry.Konfable.Name()
		logo, ok := konfables.Logos[name]
		if !ok {
			t.Errorf("missing logo for %s", name)
			continue
		}
		if logo.Width != 16 || logo.Height != 12 {
			t.Errorf("%s logo size = %dx%d, want 16x12", name, logo.Width, logo.Height)
		}
		if len(logo.Pixels) != logo.Height {
			t.Errorf("%s logo row count = %d, want %d", name, len(logo.Pixels), logo.Height)
			continue
		}
		for row, pixels := range logo.Pixels {
			if len(pixels) != logo.Width {
				t.Errorf("%s logo row %d width = %d, want %d", name, row, len(pixels), logo.Width)
			}
		}
	}
}

func TestPlainIconsForNonNerdFontMode(t *testing.T) {
	want := map[string]string{
		"alacritty":     "🖥️",
		"dconf":         "⚙️",
		"fuzzel":        "🔎",
		"git":           "🔀",
		"gtk":           "🎨",
		"pacman":        "📦",
		"powerlevel10k": "⚡",
		"rio":           "💻",
		"sshd":          "🖥️",
		"tmux":          "🪟",
		"waybar":        "📊",
		"yazi":          "📁",
	}

	seen := make(map[string]bool, len(want))
	for _, entry := range AllKonfablesWithInfo() {
		app, ok := entry.Konfable.(konfables.Konfable)
		if !ok {
			t.Errorf("%s does not satisfy full konfable interface", entry.Konfable.Name())
			continue
		}
		info := app.Info()
		icon, ok := want[info.Name]
		if !ok {
			continue
		}
		seen[info.Name] = true
		if info.Icon != icon {
			t.Errorf("%s plain icon = %q, want %q", info.Name, info.Icon, icon)
		}
	}

	for name := range want {
		if !seen[name] {
			t.Errorf("missing registered konfable %s", name)
		}
	}
}
