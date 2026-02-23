package pkg

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PixelArt holds a pixel grid that renders via the half-block technique.
// Height must be even. Pixel values are 256-color indices; 0 = transparent.
type PixelArt struct {
	Width, Height int
	Pixels        [][]uint8
}

// Render produces a string using ▀/▄ half-block characters.
// each terminal row encodes two pixel rows: fg=top, bg=bottom.
func (p PixelArt) Render() string {
	var b strings.Builder
	for y := 0; y < p.Height; y += 2 {
		if y > 0 {
			b.WriteByte('\n')
		}
		for x := 0; x < p.Width; x++ {
			top := p.Pixels[y][x]
			var bottom uint8
			if y+1 < p.Height {
				bottom = p.Pixels[y+1][x]
			}

			switch {
			case top != 0 && bottom != 0:
				s := lipgloss.NewStyle().
					Foreground(color256(top)).
					Background(color256(bottom))
				b.WriteString(s.Render("▀"))
			case top != 0:
				s := lipgloss.NewStyle().Foreground(color256(top))
				b.WriteString(s.Render("▀"))
			case bottom != 0:
				s := lipgloss.NewStyle().Foreground(color256(bottom))
				b.WriteString(s.Render("▄"))
			default:
				b.WriteByte(' ')
			}
		}
	}
	return b.String()
}

func color256(idx uint8) lipgloss.Color {
	return lipgloss.Color(strconv.Itoa(int(idx)))
}
