package ui

import (
	"strings"
	"testing"

	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"
)

func TestRenderFieldValueBoolUsesTextOnly(t *testing.T) {
	c := &content{theme: testTheme()}
	f := pkg.Field{Type: "bool"}

	for _, tc := range []struct {
		name      string
		value     string
		isDefault bool
	}{
		{name: "default false", value: "false", isDefault: true},
		{name: "configured true", value: "true", isDefault: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := stripANSI(c.renderFieldValue(f, tc.value, tc.isDefault))
			if got != tc.value {
				t.Fatalf("renderFieldValue() = %q, want %q", got, tc.value)
			}
			if strings.ContainsAny(got, "●○") {
				t.Fatalf("bool field value should not render a status dot: %q", got)
			}
		})
	}
}

func TestRenderFieldValueBlocklistShowsSummary(t *testing.T) {
	c := &content{theme: testTheme()}
	f := pkg.Field{Widget: "blocklist", Type: "string"}

	cfg := "Host web\n    User git\nMatch host bastion\n    ForwardAgent yes\n"
	enc := pkg.Encode(pkg.Parse([]byte(cfg), []string{"Host", "Match"}, nil))

	got := stripANSI(c.renderFieldValue(f, enc, false))
	if !strings.Contains(got, "Host web") || !strings.Contains(got, "Match host bastion") {
		t.Fatalf("blocklist summary = %q, want it to mention block headers", got)
	}
	// must not leak the opaque encoding (tags, byte-lengths, embedded newlines).
	if strings.ContainsAny(got, "\n") {
		t.Fatalf("blocklist summary should be single-line, got %q", got)
	}
}

func TestSplitWidthsGivesDetailMoreContentArea(t *testing.T) {
	c := &content{
		schema: &pkg.Schema{},
		config: &pkg.ConfigFile{},
		fields: []pkg.Field{{Key: "field"}},
	}

	fieldW, detailW := c.splitWidths(100)
	if fieldW != 55 || detailW != 45 {
		t.Fatalf("splitWidths(100) = (%d, %d), want (55, 45)", fieldW, detailW)
	}

	fieldW, detailW = c.splitWidths(50)
	if fieldW != 30 || detailW != 20 {
		t.Fatalf("splitWidths(50) = (%d, %d), want minimum field/detail split (30, 20)", fieldW, detailW)
	}
}

func TestRenderFieldValueColorShowsHexWithoutMarker(t *testing.T) {
	c := &content{theme: testTheme()}
	f := pkg.Field{Type: "color"}

	got := stripANSI(c.renderFieldValue(f, "aabbcc", false))
	if got != "#aabbcc" {
		t.Fatalf("renderFieldValue() = %q, want %q", got, "#aabbcc")
	}
	if strings.Contains(got, "##") {
		t.Fatalf("color field value should not render a ## marker: %q", got)
	}
	if strings.Contains(got, "██") {
		t.Fatalf("color field value should not render block swatches: %q", got)
	}
}

func TestRenderFieldValueColorKeepsHyprlandARGB(t *testing.T) {
	c := &content{theme: testTheme()}
	f := pkg.Field{Type: "color"}

	for _, value := range []string{"0xee1a1a1a", "#0xee1a1a1a"} {
		t.Run(value, func(t *testing.T) {
			got := stripANSI(c.renderFieldValue(f, value, false))
			if got != "0xee1a1a1a" {
				t.Fatalf("renderFieldValue() = %q, want %q", got, "0xee1a1a1a")
			}
			if strings.Contains(got, "#0x") {
				t.Fatalf("hyprland argb color should not be displayed as rgb hex: %q", got)
			}
		})
	}
}

func TestColorRenderHexParsesAlphaFormats(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "bare rgb", value: "aabbcc", want: "#aabbcc"},
		{name: "hash rgb", value: "#aabbcc", want: "#aabbcc"},
		{name: "hash rgba", value: "#aabbccdd", want: "#aabbcc"},
		{name: "hyprland argb", value: "0xee1a1a1a", want: "#1a1a1a"},
		{name: "legacy prefixed argb", value: "#0xee1a1a1a", want: "#1a1a1a"},
		{name: "hyprland rgb", value: "rgb(33ccff)", want: "#33ccff"},
		{name: "hyprland rgba", value: "rgba(33ccffee)", want: "#33ccff"},
		{name: "hyprland gradient", value: "rgba(33ccffee) rgba(00ff99ee) 45deg", want: "#33ccff"},
		{name: "named color", value: "bright", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := theme.ColorRenderHex(tt.value); got != tt.want {
				t.Fatalf("colorRenderHex(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestFormatPaletteColorPreservesHyprlandAlpha(t *testing.T) {
	tests := []struct {
		name     string
		template string
		selected string
		want     string
	}{
		{name: "argb", template: "0xee1a1a1a", selected: "#cba6f7", want: "0xeecba6f7"},
		{name: "rgba", template: "rgba(1a1a1aee)", selected: "#cba6f7", want: "rgba(cba6f7ee)"},
		{name: "rgb", template: "rgb(1a1a1a)", selected: "#cba6f7", want: "rgb(cba6f7)"},
		{name: "gradient unchanged", template: "rgba(1a1a1aee) rgba(333333ee) 45deg", selected: "#cba6f7", want: "#cba6f7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := theme.FormatPaletteColor(tt.template, tt.selected); got != tt.want {
				t.Fatalf("formatPaletteColor(%q, %q) = %q, want %q", tt.template, tt.selected, got, tt.want)
			}
		})
	}
}

