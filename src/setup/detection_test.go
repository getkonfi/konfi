package setup

import (
	"testing"

	"github.com/eminert/konfi/konfables"
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
