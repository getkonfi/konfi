package konfables

import "github.com/emin/konfigurator/pkg"

// Logos maps app names to their pixel art representations.
// each logo is 16×12 pixels using 256-color indices (0 = transparent).
var Logos = map[string]pkg.PixelArt{
	"ghostty":   ghosttyLogo,
	"alacritty": alacrittyLogo,
	"starship":  starshipLogo,
	"hyprland":  hyprlandLogo,
}

// color aliases for readability
const (
	__ uint8 = 0   // transparent
	wh uint8 = 255 // white
	lg uint8 = 252 // light gray
	dk uint8 = 236 // dark (eyes/details)
	or uint8 = 208 // orange
	yl uint8 = 214 // yellow
	cy uint8 = 45  // cyan
	bl uint8 = 39  // blue
	rd uint8 = 203 // red (flame)
	lb uint8 = 75  // light blue
	mb uint8 = 111 // medium blue
)

// ghostty — ghost silhouette with eyes, wavy bottom
var ghosttyLogo = pkg.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, __, __, wh, wh, wh, wh, wh, wh, __, __, __, __, __},
		{__, __, __, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, __, __, __},
		{__, __, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, __, __},
		{__, __, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, __, __},
		{__, __, wh, wh, dk, dk, wh, wh, wh, wh, dk, dk, wh, wh, __, __},
		{__, __, wh, wh, dk, dk, wh, wh, wh, wh, dk, dk, wh, wh, __, __},
		{__, __, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, __, __},
		{__, __, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, __, __},
		{__, __, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, __, __},
		{__, __, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, __, __},
		{__, __, wh, __, __, wh, wh, __, __, wh, wh, __, __, wh, __, __},
		{__, __, __, __, __, __, wh, __, __, wh, __, __, __, __, __, __},
	},
}

// alacritty — triangle/A mark in orange and yellow
var alacrittyLogo = pkg.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, __, __, __, __, yl, yl, __, __, __, __, __, __, __},
		{__, __, __, __, __, __, yl, yl, yl, yl, __, __, __, __, __, __},
		{__, __, __, __, __, yl, yl, yl, yl, yl, yl, __, __, __, __, __},
		{__, __, __, __, or, yl, yl, yl, yl, yl, yl, or, __, __, __, __},
		{__, __, __, or, or, yl, yl, yl, yl, yl, yl, or, or, __, __, __},
		{__, __, or, or, or, or, or, or, or, or, or, or, or, or, __, __},
		{__, or, or, yl, yl, yl, __, __, __, __, yl, yl, yl, or, or, __},
		{__, or, yl, yl, yl, __, __, __, __, __, __, yl, yl, yl, or, __},
		{or, or, yl, yl, __, __, __, __, __, __, __, __, yl, yl, or, or},
		{or, yl, yl, __, __, __, __, __, __, __, __, __, __, yl, yl, or},
		{or, yl, __, __, __, __, __, __, __, __, __, __, __, __, yl, or},
		{or, or, __, __, __, __, __, __, __, __, __, __, __, __, or, or},
	},
}

// starship — rocket pointing upward with exhaust flame
var starshipLogo = pkg.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, __, __, __, __, cy, cy, __, __, __, __, __, __, __},
		{__, __, __, __, __, __, cy, cy, cy, cy, __, __, __, __, __, __},
		{__, __, __, __, __, cy, bl, bl, bl, bl, cy, __, __, __, __, __},
		{__, __, __, __, __, cy, bl, bl, bl, bl, cy, __, __, __, __, __},
		{__, __, __, __, __, bl, bl, bl, bl, bl, bl, __, __, __, __, __},
		{__, __, __, __, cy, bl, bl, bl, bl, bl, bl, cy, __, __, __, __},
		{__, __, __, cy, cy, bl, bl, bl, bl, bl, bl, cy, cy, __, __, __},
		{__, __, cy, __, __, bl, bl, bl, bl, bl, bl, __, __, cy, __, __},
		{__, cy, __, __, __, bl, bl, bl, bl, bl, bl, __, __, __, cy, __},
		{__, __, __, __, __, __, bl, bl, bl, bl, __, __, __, __, __, __},
		{__, __, __, __, __, __, rd, rd, rd, rd, __, __, __, __, __, __},
		{__, __, __, __, __, __, __, rd, rd, __, __, __, __, __, __, __},
	},
}

// hyprland — abstract flowing wave/swirl
var hyprlandLogo = pkg.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, __, lb, lb, lb, lb, __, __, __, __, __, __, __, __},
		{__, __, __, lb, lb, mb, mb, lb, lb, __, __, __, __, __, __, __},
		{__, __, lb, lb, __, __, mb, mb, lb, lb, __, __, __, __, __, __},
		{__, __, lb, __, __, __, __, mb, mb, lb, lb, lb, __, __, __, __},
		{__, __, __, __, __, __, __, __, mb, mb, mb, lb, lb, __, __, __},
		{__, __, __, __, __, __, __, __, __, __, mb, __, __, lb, __, __},
		{__, __, lb, __, __, mb, __, __, __, __, __, __, __, __, __, __},
		{__, __, lb, lb, mb, mb, mb, __, __, __, __, __, __, lb, __, __},
		{__, __, __, __, lb, lb, mb, mb, __, __, __, __, lb, lb, __, __},
		{__, __, __, __, __, lb, lb, mb, mb, __, __, lb, lb, __, __, __},
		{__, __, __, __, __, __, lb, lb, mb, mb, lb, lb, __, __, __, __},
		{__, __, __, __, __, __, __, __, lb, lb, lb, __, __, __, __, __},
	},
}
