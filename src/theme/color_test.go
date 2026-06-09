package theme

import "testing"

func TestFormatPaletteColorPreservesBareRGBA(t *testing.T) {
	tests := []struct {
		name     string
		template string
		selected string
		want     string
	}{
		{name: "bare rgba keeps alpha and no hash", template: "fdf6e3ff", selected: "#1e1e2e", want: "1e1e2eff"},
		{name: "bare rgba selected keeps notation", template: "fdf6e3ff", selected: "313244ff", want: "313244ff"},
		{name: "bare rgb keeps no hash", template: "fdf6e3", selected: "#1e1e2e", want: "1e1e2e"},
		{name: "hash rgba keeps alpha and hash", template: "#fdf6e3cc", selected: "#1e1e2e", want: "#1e1e2ecc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatPaletteColor(tt.template, tt.selected); got != tt.want {
				t.Fatalf("FormatPaletteColor(%q, %q) = %q, want %q", tt.template, tt.selected, got, tt.want)
			}
		})
	}
}
