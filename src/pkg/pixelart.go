package pkg

import (
	"image/color"
	"math"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
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

func color256(idx uint8) color.Color {
	return lipgloss.Color(strconv.Itoa(int(idx)))
}

// Clone returns a deep copy of the pixel grid.
func (p PixelArt) Clone() PixelArt {
	c := PixelArt{Width: p.Width, Height: p.Height, Pixels: make([][]uint8, len(p.Pixels))}
	for i, row := range p.Pixels {
		c.Pixels[i] = make([]uint8, len(row))
		copy(c.Pixels[i], row)
	}
	return c
}

// AnimKind identifies the type of logo animation.
type AnimKind int

const (
	AnimNone  AnimKind = iota
	AnimBlink          // ghostty eye blink
	AnimFlame          // starship exhaust fire
	AnimFade           // alacritty radial fade-in
	AnimWave           // hyprland color wave
)

// Pixel identifies a single pixel coordinate.
type Pixel struct{ Row, Col int }

// AnimConfig holds per-app animation parameters.
type AnimConfig struct {
	Kind   AnimKind
	Frames int // total frames
	TickMs int // ms per frame (60 for 60ms)

	// blink-specific
	BlinkPixels []Pixel  // eye pixel coordinates
	BlinkSeq    []bool   // per-frame: true = eyes visible

	// flame-specific
	FlameZone   [4]int   // {rowMin, rowMax, colMin, colMax}
	FlameColors []uint8  // particle colors (bright→dim)
	FlameRamp   []int    // per-frame spawn count envelope

	// wave-specific
	WaveBright []uint8 // brightness color ramp (bright→base)
}

// Particle is a short-lived colored pixel for the flame effect.
type Particle struct {
	Row, Col       int
	Color          uint8
	Life, MaxLife  int
}

// AnimState holds the mutable state of a running animation.
type AnimState struct {
	Config    AnimConfig
	Base      PixelArt
	Frame     int
	Done      bool
	Particles []Particle
	rng       uint64
}

// NewAnimState creates a fresh animation state.
func NewAnimState(base PixelArt, cfg AnimConfig) *AnimState {
	return &AnimState{
		Config: cfg,
		Base:   base.Clone(),
		rng:    0x5DEECE66D, // seed
	}
}

// xorshift64 inline PRNG
func (s *AnimState) rand() uint64 {
	s.rng ^= s.rng << 13
	s.rng ^= s.rng >> 7
	s.rng ^= s.rng << 17
	return s.rng
}

// Tick advances the animation by one frame. returns true when finished.
func (s *AnimState) Tick() bool {
	if s.Done {
		return true
	}
	s.Frame++
	if s.Frame >= s.Config.Frames {
		s.Done = true
		return true
	}
	return false
}

// CurrentFrame produces the pixel art for the current animation frame.
func (s *AnimState) CurrentFrame() PixelArt {
	frame := s.Base.Clone()
	switch s.Config.Kind {
	case AnimBlink:
		s.applyBlink(&frame)
	case AnimFlame:
		s.applyFlame(&frame)
	case AnimFade:
		s.applyFade(&frame)
	case AnimWave:
		s.applyWave(&frame)
	}
	return frame
}

// applyBlink toggles eye pixels between dark and transparent per BlinkSeq.
func (s *AnimState) applyBlink(frame *PixelArt) {
	idx := s.Frame
	if idx >= len(s.Config.BlinkSeq) {
		return
	}
	if !s.Config.BlinkSeq[idx] {
		// eyes closed: set eye pixels to transparent
		for _, p := range s.Config.BlinkPixels {
			if p.Row >= 0 && p.Row < frame.Height && p.Col >= 0 && p.Col < frame.Width {
				frame.Pixels[p.Row][p.Col] = 0
			}
		}
	}
}

// applyFlame spawns, ages, and renders particles in the exhaust zone.
func (s *AnimState) applyFlame(frame *PixelArt) {
	zone := s.Config.FlameZone
	colors := s.Config.FlameColors
	if len(colors) == 0 {
		return
	}

	// spawn new particles based on ramp envelope
	spawnCount := 0
	if s.Frame < len(s.Config.FlameRamp) {
		spawnCount = s.Config.FlameRamp[s.Frame]
	}
	for i := 0; i < spawnCount; i++ {
		r := zone[0] + int(s.rand()%uint64(zone[1]-zone[0]+1))
		c := zone[2] + int(s.rand()%uint64(zone[3]-zone[2]+1))
		maxLife := 3 + int(s.rand()%4)
		s.Particles = append(s.Particles, Particle{
			Row: r, Col: c,
			Color:   colors[0],
			Life:    0,
			MaxLife: maxLife,
		})
	}

	// age particles
	alive := s.Particles[:0]
	for i := range s.Particles {
		p := &s.Particles[i]
		p.Life++
		if p.Life >= p.MaxLife {
			continue
		}
		// drift upward occasionally
		if p.Life > 1 && s.rand()%3 == 0 {
			p.Row--
		}
		// kill particles that escaped above the flame zone
		if p.Row < zone[0] {
			continue
		}
		// color fades through the palette over lifetime
		ci := p.Life * len(colors) / p.MaxLife
		if ci >= len(colors) {
			ci = len(colors) - 1
		}
		p.Color = colors[ci]
		alive = append(alive, *p)
	}
	s.Particles = alive

	// render particles onto frame
	for _, p := range s.Particles {
		if p.Row >= 0 && p.Row < frame.Height && p.Col >= 0 && p.Col < frame.Width {
			frame.Pixels[p.Row][p.Col] = p.Color
		}
	}
}

// applyFade reveals pixels radially from center based on frame progress.
func (s *AnimState) applyFade(frame *PixelArt) {
	cx := float64(frame.Width) / 2.0
	cy := float64(frame.Height) / 2.0
	maxDist := math.Sqrt(cx*cx + cy*cy)
	progress := float64(s.Frame) / float64(s.Config.Frames)
	threshold := progress * maxDist

	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			dx := float64(x) - cx + 0.5
			dy := float64(y) - cy + 0.5
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > threshold {
				frame.Pixels[y][x] = 0 // hide pixels beyond wavefront
			}
		}
	}
}

// applyWave sends a brightness wavefront outward from center.
func (s *AnimState) applyWave(frame *PixelArt) {
	if len(s.Config.WaveBright) == 0 {
		return
	}

	cx := float64(frame.Width) / 2.0
	cy := float64(frame.Height) / 2.0
	maxDist := math.Sqrt(cx*cx + cy*cy)
	// wavefront position moves outward over time
	wavePos := float64(s.Frame) / float64(s.Config.Frames) * maxDist * 1.3
	waveWidth := 3.0

	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			if s.Base.Pixels[y][x] == 0 {
				continue // skip transparent base pixels
			}
			dx := float64(x) - cx + 0.5
			dy := float64(y) - cy + 0.5
			dist := math.Sqrt(dx*dx + dy*dy)
			// how close is this pixel to the wavefront?
			delta := math.Abs(dist - wavePos)
			if delta < waveWidth {
				// pixel is in the wave band — apply brightness ramp
				t := delta / waveWidth
				ci := int(t * float64(len(s.Config.WaveBright)-1))
				if ci >= len(s.Config.WaveBright) {
					ci = len(s.Config.WaveBright) - 1
				}
				frame.Pixels[y][x] = s.Config.WaveBright[ci]
			}
		}
	}
}
