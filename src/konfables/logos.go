package konfables

import "github.com/eminert/konfi/pkg/pixelart"

// Logos maps app names to their pixel art representations.
// each logo is 16×12 pixels using 256-color indices (0 = transparent).
var Logos = map[string]pixelart.PixelArt{
	"ghostty":       ghosttyLogo,
	"alacritty":     alacrittyLogo,
	"starship":      starshipLogo,
	"powerlevel10k": powerlevel10kLogo,
	"hyprland":      hyprlandLogo,
	"fuzzel":        fuzzelLogo,
	"waybar":        waybarLogo,
	"konfi":         konfiLogo,
	"git":           gitLogo,
	"tmux":          tmuxLogo,
	"ssh":           sshLogo,
	"sshd":          sshLogo,
	"pacman":        pacmanLogo,
	"dconf":         dconfLogo,
	"gnome":         gnomeLogo,
	"kitty":         kittyLogo,
	"helix":         helixLogo,
	"yazi":          yaziLogo,
	"rio":           rioLogo,
	"gtk":           gtkLogo,
	"brew":          brewLogo,
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

// powerlevel10k — prompt chevrons with lightning bolt
var powerlevel10kLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, pu, pu, __, __, __, __, yl, yl, __, __, __, cy, cy, __},
		{__, __, __, pu, pu, __, __, yl, yl, __, __, __, cy, cy, __, __},
		{__, __, __, __, pu, pu, yl, yl, __, __, __, cy, cy, __, __, __},
		{__, __, __, __, __, pu, yl, yl, yl, yl, cy, cy, __, __, __, __},
		{__, __, __, __, pu, pu, __, yl, yl, cy, cy, __, __, __, __, __},
		{__, __, __, pu, pu, __, __, yl, yl, __, cy, cy, __, __, __, __},
		{__, __, pu, pu, __, __, yl, yl, __, __, __, cy, cy, __, __, __},
		{__, pu, pu, __, __, yl, yl, __, __, __, __, __, cy, cy, __, __},
		{__, __, pu, pu, __, __, yl, yl, __, __, __, cy, cy, __, __, __},
		{__, __, __, pu, pu, __, __, yl, yl, __, cy, cy, __, __, __, __},
		{__, __, __, __, pu, pu, __, yl, yl, cy, cy, __, __, __, __, __},
		{__, __, __, __, __, pu, pu, yl, cy, cy, __, __, __, __, __, __},
	},
}

// konfi — gear/cog icon
var konfiLogo = pixelart.PixelArt{
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
	"powerlevel10k": sequenceAnim(
		18, 70, []uint8{wh, lg},
		[]int{-1, 0, 0, 1, 1, 2, 2, 1, 1, -1, -1, 1, -1, -1, -1, -1, -1, -1},
		logoColorPixels(powerlevel10kLogo, pu),
		logoColorPixels(powerlevel10kLogo, yl),
		logoColorPixels(powerlevel10kLogo, cy),
	),
	"alacritty": {
		Kind: pixelart.AnimFade, Frames: 12, TickMs: 60,
	},
	"hyprland": {
		Kind: pixelart.AnimDrip, Frames: 33, TickMs: 60,
		DripOrigin: pixelart.Pixel{Row: 0, Col: 7},
		DripBright: []uint8{255, 195, 153, 111},
	},
	"fuzzel": sequenceAnim(
		20, 65, []uint8{wh, lg},
		[]int{-1, 0, 0, 1, 2, 3, 4, 3, 4, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		logoColorPixels(fuzzelLogo, cy, or),
		logoRectPixels(fuzzelLogo, 9, 9, 0, 15, lg),
		logoRectPixels(fuzzelLogo, 10, 10, 0, 15, lg),
		logoRectPixels(fuzzelLogo, 11, 11, 0, 15, lg),
		logoRectPixels(fuzzelLogo, 10, 11, 8, 12, lg),
	),
	"waybar": sequenceAnim(
		22, 60, []uint8{wh, lg},
		[]int{-1, 0, 1, 2, 3, 4, 5, 4, 3, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		logoRectPixels(waybarLogo, 2, 3, 1, 3, cy),
		logoRectPixels(waybarLogo, 2, 3, 6, 8, gn),
		logoRectPixels(waybarLogo, 2, 3, 11, 13, yl),
		logoRectPixels(waybarLogo, 5, 10, 1, 6, cy),
		logoRectPixels(waybarLogo, 5, 10, 7, 8, gn),
		logoRectPixels(waybarLogo, 5, 10, 9, 14, yl),
	),
	"konfi": sequenceAnim(
		20, 70, []uint8{wh, lg},
		[]int{-1, 0, 1, 2, 3, 4, 4, 3, 2, 1, 0, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		logoRectPixels(konfiLogo, 0, 2, 4, 11, lg),
		logoRectPixels(konfiLogo, 3, 8, 10, 14, lg),
		logoRectPixels(konfiLogo, 9, 11, 4, 11, lg),
		logoRectPixels(konfiLogo, 3, 8, 1, 5, lg),
		logoRectPixels(konfiLogo, 2, 9, 5, 10, dk),
	),
	"git": sequenceAnim(
		20, 65, []uint8{wh, yl},
		[]int{-1, 0, 1, 2, 3, 4, 3, 2, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		logoRectPixels(gitLogo, 0, 2, 5, 10, or),
		logoRectPixels(gitLogo, 3, 5, 2, 6, or),
		logoRectPixels(gitLogo, 5, 6, 7, 8, or),
		logoRectPixels(gitLogo, 3, 7, 9, 13, or),
		logoRectPixels(gitLogo, 8, 10, 6, 9, or),
	),
	"tmux": sequenceAnim(
		22, 75, []uint8{wh, lg},
		[]int{-1, 0, 1, 2, 0, 1, 2, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		logoRectPixels(tmuxLogo, 1, 9, 2, 6, gn),
		logoRectPixels(tmuxLogo, 1, 4, 8, 13, gn),
		logoRectPixels(tmuxLogo, 6, 9, 8, 13, gn),
	),
	"ssh":  sshKeyAnim,
	"sshd": sshKeyAnim,
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
	"gnome": sequenceAnim(
		20, 75, []uint8{wh, lg},
		[]int{-1, 0, 1, 2, 3, 4, 4, -1, 4, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		logoRectPixels(gnomeLogo, 0, 1, 4, 5, lb),
		logoRectPixels(gnomeLogo, 0, 1, 8, 9, lb),
		logoRectPixels(gnomeLogo, 0, 1, 11, 12, lb),
		logoRectPixels(gnomeLogo, 2, 3, 5, 11, lb),
		logoRectPixels(gnomeLogo, 4, 10, 3, 11, lb),
	),
	"dconf": sequenceAnim(
		22, 65, []uint8{wh, lg},
		[]int{-1, 0, 1, 2, 3, 4, 5, 4, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		logoRectPixels(dconfLogo, 0, 1, 3, 12, bl, lb),
		logoRectPixels(dconfLogo, 2, 3, 2, 12, bl, lb),
		logoRectPixels(dconfLogo, 4, 5, 2, 12, bl, lb),
		logoRectPixels(dconfLogo, 6, 8, 2, 12, bl, lb),
		logoRectPixels(dconfLogo, 9, 11, 2, 12, bl, lb),
		logoRectPixels(dconfLogo, 6, 8, 6, 8, lg, dk),
	),
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
	"yazi": sequenceAnim(
		20, 65, []uint8{wh, lg},
		[]int{-1, 0, 1, 2, 3, 2, 1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		logoRectPixels(yaziLogo, 0, 2, 2, 13, yl),
		logoRectPixels(yaziLogo, 4, 9, 1, 4, yl),
		logoRectPixels(yaziLogo, 4, 9, 5, 8, dk, lg),
		logoRectPixels(yaziLogo, 4, 9, 9, 14, yl, dk, lg),
	),
	"gtk": sequenceAnim(
		22, 65, []uint8{wh, lg},
		[]int{-1, 0, 1, 2, 3, 4, 5, 5, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		logoColorPixels(gtkLogo, rd),
		logoColorPixels(gtkLogo, cy),
		logoColorPixels(gtkLogo, gn),
		logoColorPixels(gtkLogo, yl),
		logoColorPixels(gtkLogo, pu),
		logoColorPixels(gtkLogo, wh, or),
	),
	"brew": sequenceAnim(
		22, 75, []uint8{wh, lg},
		[]int{-1, 0, -1, 1, 2, 3, 4, 4, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		logoRectPixels(brewLogo, 0, 1, 2, 11, wh),
		logoRectPixels(brewLogo, 2, 3, 1, 13, or, yl),
		logoRectPixels(brewLogo, 4, 6, 1, 14, or, yl),
		logoRectPixels(brewLogo, 7, 9, 1, 13, or, yl),
		logoRectPixels(brewLogo, 10, 11, 2, 10, or),
	),
}

var sshKeyAnim = sequenceAnim(
	18, 70, []uint8{wh, lg},
	[]int{-1, 0, 0, 1, 2, 3, 2, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
	logoRectPixels(sshLogo, 0, 4, 5, 10, yl),
	logoRectPixels(sshLogo, 5, 6, 7, 8, yl),
	logoRectPixels(sshLogo, 7, 7, 7, 10, yl),
	logoRectPixels(sshLogo, 8, 10, 7, 9, yl),
)

func sequenceAnim(frames, tickMs int, bright []uint8, seq []int, groups ...[]pixelart.Pixel) pixelart.AnimConfig {
	return pixelart.AnimConfig{
		Kind:           pixelart.AnimSequence,
		Frames:         frames,
		TickMs:         tickMs,
		SequenceGroups: groups,
		SequenceSeq:    seq,
		SequenceBright: bright,
	}
}

func logoColorPixels(logo pixelart.PixelArt, colors ...uint8) []pixelart.Pixel {
	return logoRectPixels(logo, 0, logo.Height-1, 0, logo.Width-1, colors...)
}

func logoRectPixels(logo pixelart.PixelArt, rowMin, rowMax, colMin, colMax int, colors ...uint8) []pixelart.Pixel {
	colorSet := make(map[uint8]struct{}, len(colors))
	for _, c := range colors {
		colorSet[c] = struct{}{}
	}

	var pixels []pixelart.Pixel
	for row := rowMin; row <= rowMax && row < logo.Height; row++ {
		if row < 0 {
			continue
		}
		for col := colMin; col <= colMax && col < logo.Width; col++ {
			if col < 0 {
				continue
			}
			c := logo.Pixels[row][col]
			if c == 0 {
				continue
			}
			if len(colorSet) > 0 {
				if _, ok := colorSet[c]; !ok {
					continue
				}
			}
			pixels = append(pixels, pixelart.Pixel{Row: row, Col: col})
		}
	}
	return pixels
}

// fuzzel — launcher search lens over list rows
var fuzzelLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, __, __, __, cy, cy, cy, cy, __, __, __, __, __, __},
		{__, __, __, __, cy, cy, __, __, __, __, cy, cy, __, __, __, __},
		{__, __, __, cy, __, __, wh, wh, wh, __, __, cy, __, __, __, __},
		{__, __, cy, __, __, wh, wh, wh, wh, wh, __, __, cy, __, __, __},
		{__, __, cy, __, __, wh, wh, wh, wh, wh, __, __, cy, __, __, __},
		{__, __, cy, __, __, wh, wh, wh, wh, wh, __, __, cy, __, __, __},
		{__, __, __, cy, __, __, wh, wh, wh, __, __, cy, __, __, __, __},
		{__, __, __, __, cy, cy, __, __, __, __, cy, cy, __, __, __, __},
		{__, __, __, __, __, __, cy, cy, cy, cy, or, or, __, __, __, __},
		{__, __, lg, lg, lg, lg, lg, __, __, __, __, or, or, __, __, __},
		{__, __, lg, lg, lg, lg, lg, __, lg, lg, lg, __, or, or, __, __},
		{__, __, lg, lg, lg, lg, lg, __, lg, lg, lg, __, __, or, or, __},
	},
}

// waybar — top bar with segmented modules
var waybarLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
		{lb, lb, lb, lb, lb, lb, lb, lb, lb, lb, lb, lb, lb, lb, lb, lb},
		{lb, cy, cy, cy, lb, lb, gn, gn, gn, lb, lb, yl, yl, yl, lb, lb},
		{lb, cy, cy, cy, lb, lb, gn, gn, gn, lb, lb, yl, yl, yl, lb, lb},
		{lb, lb, lb, lb, lb, lb, lb, lb, lb, lb, lb, lb, lb, lb, lb, lb},
		{__, __, __, cy, cy, __, __, gn, gn, __, __, yl, yl, __, __, __},
		{__, __, cy, cy, cy, cy, __, gn, gn, __, yl, yl, yl, yl, __, __},
		{__, cy, cy, __, __, cy, cy, gn, gn, yl, yl, __, __, yl, yl, __},
		{__, cy, cy, __, __, cy, cy, gn, gn, yl, yl, __, __, yl, yl, __},
		{__, __, cy, cy, cy, cy, __, gn, gn, __, yl, yl, yl, yl, __, __},
		{__, __, __, cy, cy, __, __, gn, gn, __, __, yl, yl, __, __, __},
		{__, __, __, __, __, __, __, __, __, __, __, __, __, __, __, __},
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

// gnome — foot silhouette in light blue
var gnomeLogo = pixelart.PixelArt{
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

// dconf — settings database with cog teeth
var dconfLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, __, bl, bl, bl, bl, bl, bl, bl, bl, __, __, __, __},
		{__, __, __, bl, bl, lb, lb, lb, lb, lb, lb, bl, bl, __, __, __},
		{__, __, bl, lb, lb, lb, lb, lb, lb, lb, lb, lb, bl, __, __, __},
		{__, __, bl, lb, lb, lb, lb, lb, lb, lb, lb, lb, bl, __, __, __},
		{__, __, bl, lb, lb, bl, bl, bl, bl, bl, lb, lb, bl, __, __, __},
		{__, __, bl, lb, bl, bl, __, __, __, bl, bl, lb, bl, __, __, __},
		{__, __, bl, lb, bl, __, lg, lg, lg, __, bl, lb, bl, __, __, __},
		{__, __, bl, lb, bl, __, lg, dk, lg, __, bl, lb, bl, __, __, __},
		{__, __, bl, lb, bl, __, lg, lg, lg, __, bl, lb, bl, __, __, __},
		{__, __, bl, lb, bl, bl, __, __, __, bl, bl, lb, bl, __, __, __},
		{__, __, __, bl, lb, bl, bl, bl, bl, bl, lb, bl, __, __, __, __},
		{__, __, __, __, bl, bl, bl, bl, bl, bl, bl, __, __, __, __, __},
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

// yazi — split folder/file panes
var yaziLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, yl, yl, yl, yl, __, __, __, __, __, __, __, __, __, __},
		{__, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, __, __},
		{__, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, __},
		{__, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, __},
		{__, yl, yl, yl, yl, dk, dk, dk, dk, yl, yl, yl, yl, yl, yl, __},
		{__, yl, yl, yl, yl, dk, lg, lg, dk, yl, yl, yl, yl, yl, yl, __},
		{__, yl, yl, yl, yl, dk, lg, lg, dk, yl, yl, dk, dk, yl, yl, __},
		{__, yl, yl, yl, yl, dk, dk, dk, dk, yl, yl, dk, lg, yl, yl, __},
		{__, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, dk, lg, yl, yl, __},
		{__, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, dk, dk, yl, yl, __},
		{__, __, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, __, __},
		{__, __, __, yl, yl, yl, yl, yl, yl, yl, yl, yl, yl, __, __, __},
	},
}

// gtk — paint brush over palette swatches
var gtkLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, __, __, __, __, wh, wh, __, __, __, __, __, __, __},
		{__, __, __, __, __, __, wh, wh, wh, wh, __, __, __, __, __, __},
		{__, __, __, __, __, wh, wh, dk, dk, wh, wh, __, __, __, __, __},
		{__, __, __, __, wh, wh, dk, dk, dk, dk, wh, wh, __, __, __, __},
		{__, __, __, or, or, dk, dk, dk, dk, dk, dk, or, or, __, __, __},
		{__, __, or, or, dk, dk, dk, dk, dk, dk, dk, dk, or, or, __, __},
		{__, or, or, dk, dk, rd, rd, dk, cy, cy, dk, dk, dk, or, or, __},
		{__, or, dk, dk, dk, rd, rd, dk, cy, cy, dk, gn, gn, dk, or, __},
		{or, or, dk, yl, yl, dk, dk, dk, dk, dk, dk, gn, gn, dk, or, or},
		{or, dk, dk, yl, yl, dk, pu, pu, dk, dk, dk, dk, dk, dk, dk, or},
		{or, or, dk, dk, dk, dk, pu, pu, dk, dk, dk, dk, dk, dk, or, or},
		{__, or, or, or, dk, dk, dk, dk, dk, dk, dk, or, or, or, __, __},
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

// brew — mug with foam
var brewLogo = pixelart.PixelArt{
	Width: 16, Height: 12,
	Pixels: [][]uint8{
		{__, __, __, wh, wh, __, wh, wh, __, wh, wh, __, __, __, __, __},
		{__, __, wh, wh, wh, wh, wh, wh, wh, wh, wh, wh, __, __, __, __},
		{__, or, or, or, or, or, or, or, or, or, or, or, or, __, __, __},
		{__, or, yl, yl, yl, yl, yl, yl, yl, yl, yl, or, or, or, __, __},
		{__, or, yl, yl, yl, yl, yl, yl, yl, yl, yl, or, __, or, or, __},
		{__, or, yl, yl, yl, yl, yl, yl, yl, yl, yl, or, __, __, or, __},
		{__, or, yl, yl, yl, yl, yl, yl, yl, yl, yl, or, __, __, or, __},
		{__, or, yl, yl, yl, yl, yl, yl, yl, yl, yl, or, __, or, or, __},
		{__, or, yl, yl, yl, yl, yl, yl, yl, yl, yl, or, or, or, __, __},
		{__, or, yl, yl, yl, yl, yl, yl, yl, yl, yl, or, __, __, __, __},
		{__, __, or, or, or, or, or, or, or, or, or, __, __, __, __, __},
		{__, __, __, or, or, or, or, or, or, or, __, __, __, __, __, __},
	},
}
