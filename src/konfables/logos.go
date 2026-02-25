package konfables

import "github.com/emin/konfigurator/pkg"

// Logos maps app names to their pixel art representations.
// each logo is 16×12 pixels using 256-color indices (0 = transparent).
var Logos = map[string]pkg.PixelArt{
	"ghostty":      ghosttyLogo,
	"alacritty":    alacrittyLogo,
	"starship":     starshipLogo,
	"hyprland":     hyprlandLogo,
	"konfigurator": konfiguratorLogo,
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

// konfigurator — gear/cog icon
var konfiguratorLogo = pkg.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, __, __, lg, lg, __, __, lg, lg, __, __, __, __, __},
		{__, __, __, __, lg, lg, lg, lg, lg, lg, lg, lg, __, __, __, __},
		{__, __, __, __, lg, lg, dk, dk, dk, dk, lg, lg, __, __, __, __},
		{__, __, lg, lg, lg, dk, dk, __, __, dk, dk, lg, lg, lg, __, __},
		{__, lg, lg, lg, dk, dk, __, __, __, __, dk, dk, lg, lg, lg, __},
		{__, lg, lg, lg, dk, __, __, __, __, __, __, dk, lg, lg, lg, __},
		{__, lg, lg, lg, dk, __, __, __, __, __, __, dk, lg, lg, lg, __},
		{__, lg, lg, lg, dk, dk, __, __, __, __, dk, dk, lg, lg, lg, __},
		{__, __, lg, lg, lg, dk, dk, __, __, dk, dk, lg, lg, lg, __, __},
		{__, __, __, __, lg, lg, dk, dk, dk, dk, lg, lg, __, __, __, __},
		{__, __, __, __, lg, lg, lg, lg, lg, lg, lg, lg, __, __, __, __},
		{__, __, __, __, __, lg, lg, __, __, lg, lg, __, __, __, __, __},
	},
}

// LogoAnims maps app names to their logo animation configs.
var LogoAnims = map[string]pkg.AnimConfig{
	"ghostty": {
		Kind: pkg.AnimBlink, Frames: 8, TickMs: 60,
		BlinkPixels: []pkg.Pixel{
			{Row: 4, Col: 4}, {Row: 4, Col: 5}, {Row: 5, Col: 4}, {Row: 5, Col: 5},
			{Row: 4, Col: 10}, {Row: 4, Col: 11}, {Row: 5, Col: 10}, {Row: 5, Col: 11},
		},
		// open, open, open, closed, closed, open, open, open
		BlinkSeq: []bool{true, true, true, false, false, true, true, true},
	},
	"starship": {
		Kind: pkg.AnimFlame, Frames: 20, TickMs: 60,
		FlameZone:   [4]int{9, 11, 5, 10},
		FlameColors: []uint8{rd, or, yl, dk},
		// ramp up → plateau → die down
		FlameRamp: []int{
			1, 2, 3, 4, 5, 6, 6, 6, 6, 6,
			5, 5, 4, 4, 3, 3, 2, 1, 0, 0,
		},
	},
	"alacritty": {
		Kind: pkg.AnimFade, Frames: 12, TickMs: 60,
	},
	"hyprland": {
		Kind: pkg.AnimWave, Frames: 33, TickMs: 60,
		WaveBright: []uint8{255, 195, 153, 111},
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
