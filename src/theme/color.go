package theme

import (
	"image/color"
	"math"
	"regexp"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

var stylestringRe = regexp.MustCompile(`^\[(.+?)\]\((.+?)\)$`)

// ParseStyleString extracts symbol and style from "[symbol](style)".
// returns (raw, "") if the format doesn't match.
func ParseStyleString(s string) (symbol, style string) {
	m := stylestringRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return s, ""
	}
	return m[1], m[2]
}

// ComposeStyleString produces "[symbol](style)".
func ComposeStyleString(symbol, style string) string {
	return "[" + symbol + "](" + style + ")"
}

// ColorValue renders a color's hex tinted in its own color. when the tint sits
// too close to bgHex to stay legible, it adds a contrasting backdrop so the
// value remains readable. bgHex "" skips the contrast guard.
func ColorValue(hex, bgHex string) string {
	display := ColorDisplayValue(hex)
	if display == "" {
		return ""
	}
	renderHex := ColorRenderHex(hex)
	if renderHex == "" {
		return display
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(renderHex))
	if bgHex != "" && LowContrast(renderHex, bgHex) {
		style = style.Background(lipgloss.Color(contrastBackdrop(renderHex)))
	}
	return style.Render(display)
}

// hexRGB parses the rgb channels of a "#rrggbb[aa]" hex, ignoring any alpha.
func hexRGB(hex string) (r, g, b int, ok bool) {
	h := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(hex)), "#")
	if len(h) < 6 || !isHex(h[:6]) {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseInt(h[:6], 16, 64)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(v>>16) & 0xff, int(v>>8) & 0xff, int(v) & 0xff, true
}

// relLuminance returns perceptual luminance 0..1 for a hex color (0 on parse failure).
func relLuminance(hex string) float64 {
	r, g, b, ok := hexRGB(hex)
	if !ok {
		return 0
	}
	return (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 255
}

// LowContrast reports whether fg is too close to bg to read, using a WCAG-style
// luminance contrast ratio.
func LowContrast(fg, bg string) bool {
	hi, lo := relLuminance(fg), relLuminance(bg)
	if lo > hi {
		hi, lo = lo, hi
	}
	return (hi+0.05)/(lo+0.05) < 2.5
}

// ContrastRatio returns the WCAG contrast ratio between fg and bg.
func ContrastRatio(fg, bg color.Color) float64 {
	if fg == nil || bg == nil {
		return 1
	}
	hi, lo := colorLuminance(fg), colorLuminance(bg)
	if lo > hi {
		hi, lo = lo, hi
	}
	return (hi + 0.05) / (lo + 0.05)
}

// ReadableColor returns the first preferred color that is readable on bg,
// falling back to the highest-contrast option.
func ReadableColor(bg color.Color, preferred ...color.Color) color.Color {
	candidates := append(preferred, lipgloss.Color("#000000"), lipgloss.Color("#ffffff"))
	var best color.Color = lipgloss.Color("#ffffff")
	bestRatio := 0.0
	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		ratio := ContrastRatio(candidate, bg)
		if ratio >= 4.5 {
			return candidate
		}
		if ratio > bestRatio {
			best = candidate
			bestRatio = ratio
		}
	}
	return best
}

func colorLuminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return 0.2126*linearRGB(r) + 0.7152*linearRGB(g) + 0.0722*linearRGB(b)
}

func linearRGB(v uint32) float64 {
	f := float64(v) / 65535
	if f <= 0.03928 {
		return f / 12.92
	}
	return math.Pow((f+0.055)/1.055, 2.4)
}

// contrastBackdrop picks a backdrop that maximizes contrast with fg: a light
// chip for dark colors, a dark chip for light ones.
func contrastBackdrop(fg string) string {
	if relLuminance(fg) < 0.5 {
		return "#e6e6e6"
	}
	return "#1a1a1a"
}

// NormalizeHex returns the color value used for display.
func NormalizeHex(s string) string {
	return ColorDisplayValue(s)
}

func ColorDisplayValue(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "#0x") && len(lower) == 11 && isHex(lower[3:]) {
		return lower[1:]
	}
	if strings.HasPrefix(lower, "#") {
		digits := lower[1:]
		if (len(digits) == 6 || len(digits) == 8) && isHex(digits) {
			return "#" + digits
		}
		return s
	}
	if (len(lower) == 6 || len(lower) == 8) && isHex(lower) {
		return "#" + lower
	}
	return s
}

func ColorRenderHex(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if hex := colorRenderHexToken(s); hex != "" {
		return hex
	}
	if fields := strings.Fields(s); len(fields) > 0 {
		return colorRenderHexToken(fields[0])
	}
	return ""
}

func colorRenderHexToken(s string) string {
	lower := strings.ToLower(strings.TrimSpace(s))
	if lower == "" {
		return ""
	}
	if strings.HasPrefix(lower, "#0x") {
		lower = lower[1:]
	}
	if strings.HasPrefix(lower, "0x") {
		digits := lower[2:]
		if len(digits) == 8 && isHex(digits) {
			return "#" + digits[2:]
		}
		return ""
	}
	for _, prefix := range []string{"rgba(", "rgb("} {
		if !strings.HasPrefix(lower, prefix) {
			continue
		}
		closeIdx := strings.IndexByte(lower, ')')
		if closeIdx < len(prefix) {
			return ""
		}
		digits := strings.TrimSpace(lower[len(prefix):closeIdx])
		if (len(digits) == 6 || len(digits) == 8) && isHex(digits) {
			return "#" + digits[:6]
		}
		return ""
	}
	if strings.HasPrefix(lower, "#") {
		digits := lower[1:]
		if (len(digits) == 6 || len(digits) == 8) && isHex(digits) {
			return "#" + digits[:6]
		}
		return ""
	}
	if (len(lower) == 6 || len(lower) == 8) && isHex(lower) {
		return "#" + lower[:6]
	}
	return ""
}

// FormatPaletteColor rewrites template's color channels with selected's hex,
// preserving the template's notation (0x argb, rgb()/rgba(), bare hex).
func FormatPaletteColor(template, selected string) string {
	rgb := strings.TrimPrefix(ColorRenderHex(selected), "#")
	if rgb == "" {
		return selected
	}
	t := strings.TrimSpace(template)
	lower := strings.ToLower(t)
	if strings.HasPrefix(lower, "#0x") {
		lower = lower[1:]
	}
	if strings.HasPrefix(lower, "0x") {
		digits := lower[2:]
		if len(digits) == 8 && isHex(digits) {
			return "0x" + digits[:2] + rgb
		}
	}
	for _, prefix := range []string{"rgba(", "rgb("} {
		if strings.HasPrefix(lower, prefix) {
			closeIdx := strings.IndexByte(lower, ')')
			if closeIdx != len(lower)-1 {
				return selected
			}
			digits := strings.TrimSpace(lower[len(prefix):closeIdx])
			switch {
			case prefix == "rgba(" && len(digits) == 8 && isHex(digits):
				return "rgba(" + rgb + digits[6:] + ")"
			case prefix == "rgb(" && len(digits) == 6 && isHex(digits):
				return "rgb(" + rgb + ")"
			}
		}
	}
	return selected
}

func isHex(s string) bool {
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return s != ""
}
