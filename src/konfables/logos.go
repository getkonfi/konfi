package konfables

import "github.com/emin/konfigurator/pkg/pixelart"

// Logos maps app names to their pixel art representations.
// each logo is 16×12 pixels using 256-color indices (0 = transparent).
var Logos = map[string]pixelart.PixelArt{
	"ghostty":      ghosttyLogo,
	"alacritty":    alacrittyLogo,
	"starship":     starshipLogo,
	"hyprland":     hyprlandLogo,
	"konfigurator": konfiguratorLogo,
	"git":          gitLogo,
	"tmux":         tmuxLogo,
	"ssh":          sshLogo,
	"pacman":       pacmanLogo,
	"dconf":        dconfLogo,
	"claude":       claudeLogo,
	"gnome":        dconfLogo,
	"kitty":        kittyLogo,
	"helix":        helixLogo,
	"rio":          rioLogo,
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
	gn uint8 = 34  // green (tmux)
	co uint8 = 173 // coral (claude)
	sa uint8 = 180 // salmon (claude)
	br uint8 = 130 // brown (kitty)
	pu uint8 = 99  // purple (helix)
	pk uint8 = 199 // pink (rio)
)

// ghostty — ghost silhouette with eyes, wavy bottom
var ghosttyLogo = pixelart.PixelArt{
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

// alacritty — A mark in orange and yellow with crossbar
var alacrittyLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, __, __, __, __, yl, yl, __, __, __, __, __, __, __},
		{__, __, __, __, __, __, yl, yl, yl, yl, __, __, __, __, __, __},
		{__, __, __, __, __, yl, yl, yl, yl, yl, yl, __, __, __, __, __},
		{__, __, __, __, or, yl, yl, yl, yl, yl, yl, or, __, __, __, __},
		{__, __, __, or, or, yl, yl, yl, yl, yl, yl, or, or, __, __, __},
		{__, __, or, or, yl, yl, yl, __, __, yl, yl, yl, or, or, __, __},
		{__, or, or, yl, yl, yl, __, __, __, __, yl, yl, yl, or, or, __},
		{__, or, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, or, __},
		{or, or, yl, yl, __, __, __, __, __, __, __, __, yl, yl, or, or},
		{or, yl, yl, __, __, __, __, __, __, __, __, __, __, yl, yl, or},
		{or, yl, __, __, __, __, __, __, __, __, __, __, __, __, yl, or},
		{or, or, __, __, __, __, __, __, __, __, __, __, __, __, or, or},
	},
}

// starship — rocket pointing upward with exhaust flame
var starshipLogo = pixelart.PixelArt{
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
var konfiguratorLogo = pixelart.PixelArt{
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
var LogoAnims = map[string]pixelart.AnimConfig{
	"ghostty": {
		Kind: pixelart.AnimBlink, Frames: 18, TickMs: 80,
		BlinkPixels: []pixelart.Pixel{
			{Row: 4, Col: 4}, {Row: 4, Col: 5}, {Row: 5, Col: 4}, {Row: 5, Col: 5},
			{Row: 4, Col: 10}, {Row: 4, Col: 11}, {Row: 5, Col: 10}, {Row: 5, Col: 11},
		},
		BlinkColor: wh,
		// pause → single blink → eyes stay open
		BlinkSeq: []bool{
			true, true, true, true, true, true, true, true, true, true,
			false, false, false,
			true, true, true, true, true,
		},
	},
	"starship": {
		Kind: pixelart.AnimFlame, Frames: 20, TickMs: 60,
		FlameZone:   [4]int{10, 11, 6, 9},
		FlameColors: []uint8{rd, or, yl, dk},
		// ramp up → plateau → die down
		FlameRamp: []int{
			1, 2, 3, 4, 5, 6, 6, 6, 6, 6,
			5, 5, 4, 4, 3, 3, 2, 1, 0, 0,
		},
	},
	"alacritty": {
		Kind: pixelart.AnimFade, Frames: 12, TickMs: 60,
	},
	"hyprland": {
		Kind: pixelart.AnimDrip, Frames: 33, TickMs: 60,
		DripOrigin: pixelart.Pixel{Row: 0, Col: 7},
		DripBright: []uint8{255, 195, 153, 111},
	},
	"pacman": {
		Kind: pixelart.AnimChomp, Frames: 25, TickMs: 100,
		ChompColor: yl,
		ChompLayers: [][]pixelart.Pixel{
			// layer 0: col 13 (outermost edge)
			{{Row: 2, Col: 13}, {Row: 3, Col: 13}, {Row: 4, Col: 13}, {Row: 5, Col: 13}, {Row: 6, Col: 13}, {Row: 7, Col: 13}, {Row: 8, Col: 13}, {Row: 9, Col: 13}},
			// layer 1: col 12
			{{Row: 2, Col: 12}, {Row: 3, Col: 12}, {Row: 4, Col: 12}, {Row: 5, Col: 12}, {Row: 6, Col: 12}, {Row: 7, Col: 12}, {Row: 8, Col: 12}, {Row: 9, Col: 12}},
			// layer 2: col 11
			{{Row: 3, Col: 11}, {Row: 4, Col: 11}, {Row: 5, Col: 11}, {Row: 6, Col: 11}, {Row: 7, Col: 11}, {Row: 8, Col: 11}},
			// layer 3: col 10
			{{Row: 4, Col: 10}, {Row: 5, Col: 10}, {Row: 6, Col: 10}, {Row: 7, Col: 10}},
			// layer 4: col 9
			{{Row: 5, Col: 9}, {Row: 6, Col: 9}, {Row: 7, Col: 9}},
			// layer 5: col 8 (innermost tip)
			{{Row: 6, Col: 8}},
		},
		// open → straight-close → hold → straight-open
		ChompSeq: []int{
			0, 0, 0, 0, 0,
			1, 2, 3, 4, 5, 6,
			6, 6, 6,
			5, 4, 3, 2, 1, 0,
			0, 0, 0, 0, 0,
		},
	},
	"kitty": {
		Kind: pixelart.AnimBlink, Frames: 18, TickMs: 80,
		BlinkPixels: []pixelart.Pixel{
			{Row: 3, Col: 3}, {Row: 3, Col: 4},
			{Row: 3, Col: 11}, {Row: 3, Col: 12},
		},
		BlinkColor: br,
		// slow cat blink
		BlinkSeq: []bool{
			true, true, true, true, true, true, true, true, true, true, true, true,
			false, false, false, false, false,
			true,
		},
	},
	"helix": {
		Kind: pixelart.AnimWave, Frames: 33, TickMs: 60,
		WaveBright: []uint8{255, 195, 153, 111},
	},
	"rio": {
		Kind: pixelart.AnimFade, Frames: 12, TickMs: 60,
	},
}

// hyprland — water drop with glossy highlight
var hyprlandLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, __, __, __, __, lb, lb, __, __, __, __, __, __, __},
		{__, __, __, __, __, __, lb, mb, mb, lb, __, __, __, __, __, __},
		{__, __, __, __, __, lb, mb, mb, mb, mb, lb, __, __, __, __, __},
		{__, __, __, __, lb, mb, mb, mb, mb, mb, mb, lb, __, __, __, __},
		{__, __, __, lb, lb, mb, mb, mb, mb, mb, mb, lb, lb, __, __, __},
		{__, __, __, lb, mb, mb, wh, wh, mb, mb, mb, mb, lb, __, __, __},
		{__, __, lb, lb, mb, mb, wh, mb, mb, mb, mb, mb, lb, lb, __, __},
		{__, __, lb, mb, mb, mb, mb, mb, mb, mb, mb, mb, mb, lb, __, __},
		{__, __, lb, lb, mb, mb, mb, mb, mb, mb, mb, mb, lb, lb, __, __},
		{__, __, __, lb, lb, mb, mb, mb, mb, mb, mb, lb, lb, __, __, __},
		{__, __, __, __, lb, lb, lb, mb, mb, lb, lb, lb, __, __, __, __},
		{__, __, __, __, __, __, lb, lb, lb, lb, __, __, __, __, __, __},
	},
}

// git — branching diamond in orange (git logo silhouette)
var gitLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, __, __, __, __, or, or, __, __, __, __, __, __, __},
		{__, __, __, __, __, __, or, or, or, or, __, __, __, __, __, __},
		{__, __, __, __, __, or, or, or, or, or, or, __, __, __, __, __},
		{__, __, __, __, or, or, or, __, __, or, or, or, __, __, __, __},
		{__, __, __, or, or, or, __, __, __, __, or, or, or, __, __, __},
		{__, __, or, or, or, __, __, or, or, __, __, or, or, or, __, __},
		{__, __, __, or, or, or, __, or, or, __, or, or, or, __, __, __},
		{__, __, __, __, or, or, or, __, __, or, or, or, __, __, __, __},
		{__, __, __, __, __, or, or, or, or, or, or, __, __, __, __, __},
		{__, __, __, __, __, __, or, or, or, or, __, __, __, __, __, __},
		{__, __, __, __, __, __, __, or, or, __, __, __, __, __, __, __},
		{__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	},
}

// tmux — split pane layout in green
var tmuxLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, gn, gn, gn, gn, gn, gn, gn, gn, gn, gn, gn, gn, __, __},
		{__, __, gn, gn, gn, gn, gn, dk, gn, gn, gn, gn, gn, gn, __, __},
		{__, __, gn, gn, gn, gn, gn, dk, gn, gn, gn, gn, gn, gn, __, __},
		{__, __, gn, gn, gn, gn, gn, dk, gn, gn, gn, gn, gn, gn, __, __},
		{__, __, gn, gn, gn, gn, gn, dk, gn, gn, gn, gn, gn, gn, __, __},
		{__, __, gn, gn, gn, gn, gn, dk, dk, dk, dk, dk, dk, gn, __, __},
		{__, __, gn, gn, gn, gn, gn, dk, gn, gn, gn, gn, gn, gn, __, __},
		{__, __, gn, gn, gn, gn, gn, dk, gn, gn, gn, gn, gn, gn, __, __},
		{__, __, gn, gn, gn, gn, gn, dk, gn, gn, gn, gn, gn, gn, __, __},
		{__, __, gn, gn, gn, gn, gn, dk, gn, gn, gn, gn, gn, gn, __, __},
		{__, __, gn, gn, gn, gn, gn, gn, gn, gn, gn, gn, gn, gn, __, __},
		{__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	},
}

// pacman — open mouth pac-man shape in yellow
var pacmanLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, __, __, __, yl, yl, yl, yl, yl, __, __, __, __, __},
		{__, __, __, __, yl, yl, yl, yl, yl, yl, yl, yl, yl, __, __, __},
		{__, __, __, yl, yl, yl, yl, yl, yl, yl, yl, yl, __, __, __, __},
		{__, __, yl, yl, yl, yl, yl, yl, yl, yl, yl, __, __, __, __, __},
		{__, __, yl, yl, yl, dk, dk, yl, yl, yl, __, __, __, __, __, __},
		{__, __, yl, yl, yl, yl, yl, yl, yl, __, __, __, __, __, __, __},
		{__, __, yl, yl, yl, yl, yl, yl, __, __, __, __, __, __, __, __},
		{__, __, yl, yl, yl, yl, yl, yl, yl, __, __, __, __, __, __, __},
		{__, __, yl, yl, yl, yl, yl, yl, yl, yl, yl, __, __, __, __, __},
		{__, __, __, yl, yl, yl, yl, yl, yl, yl, yl, yl, __, __, __, __},
		{__, __, __, __, yl, yl, yl, yl, yl, yl, yl, yl, yl, __, __, __},
		{__, __, __, __, __, __, yl, yl, yl, yl, yl, __, __, __, __, __},
	},
}

// ssh — key shape in yellow
var sshLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, __, __, __, yl, yl, yl, yl, __, __, __, __, __, __},
		{__, __, __, __, __, yl, yl, __, __, yl, yl, __, __, __, __, __},
		{__, __, __, __, __, yl, __, __, __, __, yl, __, __, __, __, __},
		{__, __, __, __, __, yl, yl, __, __, yl, yl, __, __, __, __, __},
		{__, __, __, __, __, __, yl, yl, yl, yl, __, __, __, __, __, __},
		{__, __, __, __, __, __, __, yl, yl, __, __, __, __, __, __, __},
		{__, __, __, __, __, __, __, yl, yl, __, __, __, __, __, __, __},
		{__, __, __, __, __, __, __, yl, yl, yl, yl, __, __, __, __, __},
		{__, __, __, __, __, __, __, yl, yl, __, __, __, __, __, __, __},
		{__, __, __, __, __, __, __, yl, yl, yl, __, __, __, __, __, __},
		{__, __, __, __, __, __, __, yl, yl, __, __, __, __, __, __, __},
		{__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	},
}

// dconf — GNOME foot silhouette in light blue
var dconfLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, __, lb, lb, __, __, lb, lb, __, lb, lb, __, __, __},
		{__, __, __, __, lb, lb, __, __, lb, lb, __, lb, lb, __, __, __},
		{__, __, __, __, __, lb, lb, __, __, lb, lb, lb, __, __, __, __},
		{__, __, __, __, __, __, lb, lb, lb, lb, lb, __, __, __, __, __},
		{__, __, __, __, __, lb, lb, lb, lb, lb, lb, lb, __, __, __, __},
		{__, __, __, __, lb, lb, lb, lb, lb, lb, lb, lb, __, __, __, __},
		{__, __, __, lb, lb, lb, lb, lb, lb, lb, lb, __, __, __, __, __},
		{__, __, __, lb, lb, lb, lb, lb, lb, lb, __, __, __, __, __, __},
		{__, __, __, __, lb, lb, lb, lb, lb, lb, __, __, __, __, __, __},
		{__, __, __, __, __, lb, lb, lb, lb, __, __, __, __, __, __, __},
		{__, __, __, __, __, __, lb, lb, __, __, __, __, __, __, __, __},
		{__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	},
}

// claude — sparkle/diamond in coral and salmon (anthropic brand)
var claudeLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, __, __, __, __, co, co, __, __, __, __, __, __, __},
		{__, __, __, __, __, __, co, sa, sa, co, __, __, __, __, __, __},
		{__, __, __, __, __, co, sa, sa, sa, sa, co, __, __, __, __, __},
		{__, __, __, __, co, sa, sa, sa, sa, sa, sa, co, __, __, __, __},
		{__, __, __, co, sa, sa, sa, wh, wh, sa, sa, sa, co, __, __, __},
		{__, __, co, sa, sa, sa, wh, wh, wh, wh, sa, sa, sa, co, __, __},
		{__, __, co, sa, sa, sa, wh, wh, wh, wh, sa, sa, sa, co, __, __},
		{__, __, __, co, sa, sa, sa, wh, wh, sa, sa, sa, co, __, __, __},
		{__, __, __, __, co, sa, sa, sa, sa, sa, sa, co, __, __, __, __},
		{__, __, __, __, __, co, sa, sa, sa, sa, co, __, __, __, __, __},
		{__, __, __, __, __, __, co, sa, sa, co, __, __, __, __, __, __},
		{__, __, __, __, __, __, __, co, co, __, __, __, __, __, __, __},
	},
}

// kitty — cat peeking over terminal screen
var kittyLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, br, br, __, __, __, __, __, __, __, __, __, __, br, br, __},
		{__, br, br, br, __, __, __, __, __, __, __, __, br, br, br, __},
		{__, br, br, br, br, br, br, br, br, br, br, br, br, br, br, __},
		{__, br, br, yl, yl, br, br, br, br, br, br, yl, yl, br, br, __},
		{__, br, br, br, br, br, br, dk, br, br, br, br, br, br, br, __},
		{__, __, br, br, br, br, dk, br, br, dk, br, br, br, br, __, __},
		{__, __, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, __, __},
		{__, __, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, __, __},
		{__, __, dk, dk, dk, wh, wh, dk, dk, dk, dk, dk, dk, dk, __, __},
		{__, __, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, __, __},
		{__, __, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, __, __},
		{__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
	},
}

// helix — double helix strands in purple and cyan
var helixLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, pu, pu, __, __, __, __, __, __, cy, cy, __, __, __},
		{__, __, __, __, pu, pu, __, __, __, __, cy, cy, __, __, __, __},
		{__, __, __, __, __, pu, pu, __, __, cy, cy, __, __, __, __, __},
		{__, __, __, __, __, __, pu, cy, cy, pu, __, __, __, __, __, __},
		{__, __, __, __, __, cy, cy, __, __, pu, pu, __, __, __, __, __},
		{__, __, __, __, cy, cy, __, __, __, __, pu, pu, __, __, __, __},
		{__, __, __, cy, cy, __, __, __, __, __, __, pu, pu, __, __, __},
		{__, __, __, __, cy, cy, __, __, __, __, pu, pu, __, __, __, __},
		{__, __, __, __, __, cy, cy, __, __, pu, pu, __, __, __, __, __},
		{__, __, __, __, __, __, cy, pu, pu, cy, __, __, __, __, __, __},
		{__, __, __, __, __, pu, pu, __, __, cy, cy, __, __, __, __, __},
		{__, __, __, __, pu, pu, __, __, __, __, cy, cy, __, __, __, __},
	},
}

// rio — dark face with gradient border (pink→cyan) and white eyes
var rioLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, pk, pk, pk, pk, pk, pk, pk, pk, pk, pk, pk, pk, __, __},
		{__, pk, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, pk, __, __},
		{__, pk, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, pu, __, __},
		{__, pk, dk, dk, wh, wh, dk, dk, dk, dk, wh, wh, dk, pu, __, __},
		{__, pk, dk, dk, wh, dk, dk, dk, dk, dk, dk, wh, dk, pu, __, __},
		{__, pk, dk, dk, wh, wh, dk, dk, dk, dk, wh, wh, dk, cy, __, __},
		{__, pu, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, cy, __, __},
		{__, pu, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, cy, __, __},
		{__, pu, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, cy, __, __},
		{__, cy, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, dk, cy, __, __},
		{__, cy, cy, dk, dk, dk, dk, dk, dk, dk, dk, dk, cy, cy, __, __},
		{__, __, cy, cy, cy, cy, cy, cy, cy, cy, cy, cy, cy, __, __, __},
	},
}
