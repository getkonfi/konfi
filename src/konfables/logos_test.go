package konfables

import (
	"testing"

	"github.com/eminert/konfi/pkg/pixelart"
)

func TestLogoAnimsCoverLogos(t *testing.T) {
	for name := range Logos {
		if _, ok := LogoAnims[name]; !ok {
			t.Errorf("missing animation for logo %q", name)
		}
	}
	for name := range LogoAnims {
		if _, ok := Logos[name]; !ok {
			t.Errorf("animation registered for missing logo %q", name)
		}
	}
}

func TestLogoPixelGridsMatchDeclaredSize(t *testing.T) {
	for name, logo := range Logos {
		if len(logo.Pixels) != logo.Height {
			t.Errorf("%s logo height = %d rows, want %d", name, len(logo.Pixels), logo.Height)
			continue
		}
		for row, pixels := range logo.Pixels {
			if len(pixels) != logo.Width {
				t.Errorf("%s logo row %d width = %d, want %d", name, row, len(pixels), logo.Width)
			}
		}
	}
}

func TestLogoAnimConfigsAreSane(t *testing.T) {
	for name, cfg := range LogoAnims {
		logo, ok := Logos[name]
		if !ok {
			continue
		}
		if cfg.Kind == pixelart.AnimNone {
			t.Errorf("%s animation kind is AnimNone", name)
		}
		if cfg.Frames <= 0 || cfg.Frames > 120 {
			t.Errorf("%s animation frames = %d, want 1..120", name, cfg.Frames)
		}
		if cfg.TickMs <= 0 || cfg.TickMs > 1000 {
			t.Errorf("%s animation tick = %d, want 1..1000", name, cfg.TickMs)
		}

		switch cfg.Kind {
		case pixelart.AnimBlink:
			validateBlinkAnim(t, name, logo, cfg)
		case pixelart.AnimFlame:
			validateFlameAnim(t, name, logo, cfg)
		case pixelart.AnimFade:
		case pixelart.AnimWave:
			if len(cfg.WaveBright) == 0 {
				t.Errorf("%s wave animation has no bright colors", name)
			}
		case pixelart.AnimChomp:
			validateChompAnim(t, name, logo, cfg)
		case pixelart.AnimDrip:
			if len(cfg.DripBright) == 0 {
				t.Errorf("%s drip animation has no bright colors", name)
			}
			validatePixelInBounds(t, name, logo, cfg.DripOrigin)
		case pixelart.AnimSequence:
			validateSequenceAnim(t, name, logo, cfg)
		default:
			t.Errorf("%s animation has unknown kind %d", name, cfg.Kind)
		}
	}
}

func TestExistingLogoAnimsRemainStable(t *testing.T) {
	tests := map[string]struct {
		kind   pixelart.AnimKind
		frames int
		tickMs int
	}{
		"ghostty":   {kind: pixelart.AnimBlink, frames: 18, tickMs: 80},
		"starship":  {kind: pixelart.AnimFlame, frames: 20, tickMs: 60},
		"alacritty": {kind: pixelart.AnimFade, frames: 12, tickMs: 60},
		"hyprland":  {kind: pixelart.AnimSequence, frames: 24, tickMs: 60},
		"pacman":    {kind: pixelart.AnimChomp, frames: 25, tickMs: 100},
		"kitty":     {kind: pixelart.AnimBlink, frames: 18, tickMs: 80},
		"helix":     {kind: pixelart.AnimWave, frames: 33, tickMs: 60},
		"rio":       {kind: pixelart.AnimFade, frames: 12, tickMs: 60},
	}

	for name, want := range tests {
		cfg, ok := LogoAnims[name]
		if !ok {
			t.Fatalf("missing existing animation for %s", name)
		}
		if cfg.Kind != want.kind || cfg.Frames != want.frames || cfg.TickMs != want.tickMs {
			t.Errorf("%s animation = kind %d frames %d tick %d, want kind %d frames %d tick %d",
				name, cfg.Kind, cfg.Frames, cfg.TickMs, want.kind, want.frames, want.tickMs)
		}
	}
}

func validateBlinkAnim(t *testing.T, name string, logo pixelart.PixelArt, cfg pixelart.AnimConfig) {
	t.Helper()
	if len(cfg.BlinkPixels) == 0 {
		t.Errorf("%s blink animation has no pixels", name)
	}
	if len(cfg.BlinkSeq) == 0 || len(cfg.BlinkSeq) > cfg.Frames {
		t.Errorf("%s blink sequence length = %d, frames = %d", name, len(cfg.BlinkSeq), cfg.Frames)
	}
	for _, p := range cfg.BlinkPixels {
		validateVisiblePixel(t, name, logo, p)
	}
}

func validateFlameAnim(t *testing.T, name string, logo pixelart.PixelArt, cfg pixelart.AnimConfig) {
	t.Helper()
	if len(cfg.FlameColors) == 0 {
		t.Errorf("%s flame animation has no colors", name)
	}
	if len(cfg.FlameRamp) == 0 || len(cfg.FlameRamp) > cfg.Frames {
		t.Errorf("%s flame ramp length = %d, frames = %d", name, len(cfg.FlameRamp), cfg.Frames)
	}
	zone := cfg.FlameZone
	validatePixelInBounds(t, name, logo, pixelart.Pixel{Row: zone[0], Col: zone[2]})
	validatePixelInBounds(t, name, logo, pixelart.Pixel{Row: zone[1], Col: zone[3]})
}

func validateChompAnim(t *testing.T, name string, logo pixelart.PixelArt, cfg pixelart.AnimConfig) {
	t.Helper()
	if cfg.ChompColor == 0 {
		t.Errorf("%s chomp animation uses transparent fill", name)
	}
	if len(cfg.ChompLayers) == 0 {
		t.Errorf("%s chomp animation has no layers", name)
	}
	if len(cfg.ChompSeq) == 0 || len(cfg.ChompSeq) > cfg.Frames {
		t.Errorf("%s chomp sequence length = %d, frames = %d", name, len(cfg.ChompSeq), cfg.Frames)
	}
	for layer, pixels := range cfg.ChompLayers {
		if len(pixels) == 0 {
			t.Errorf("%s chomp layer %d is empty", name, layer)
		}
		for _, p := range pixels {
			validatePixelInBounds(t, name, logo, p)
		}
	}
	for frame, nLayers := range cfg.ChompSeq {
		if nLayers < 0 || nLayers > len(cfg.ChompLayers) {
			t.Errorf("%s chomp seq frame %d = %d, want 0..%d", name, frame, nLayers, len(cfg.ChompLayers))
		}
	}
}

func validateSequenceAnim(t *testing.T, name string, logo pixelart.PixelArt, cfg pixelart.AnimConfig) {
	t.Helper()
	if len(cfg.SequenceGroups) == 0 {
		t.Errorf("%s sequence animation has no groups", name)
	}
	if len(cfg.SequenceSeq) == 0 || len(cfg.SequenceSeq) > cfg.Frames {
		t.Errorf("%s sequence length = %d, frames = %d", name, len(cfg.SequenceSeq), cfg.Frames)
	}
	if len(cfg.SequenceBright) == 0 {
		t.Errorf("%s sequence animation has no bright colors", name)
	}
	for group, pixels := range cfg.SequenceGroups {
		if len(pixels) == 0 {
			t.Errorf("%s sequence group %d is empty", name, group)
		}
		for _, p := range pixels {
			validateVisiblePixel(t, name, logo, p)
		}
	}
	for frame, group := range cfg.SequenceSeq {
		if group < -1 || group >= len(cfg.SequenceGroups) {
			t.Errorf("%s sequence frame %d references group %d, want -1..%d", name, frame, group, len(cfg.SequenceGroups)-1)
		}
	}
}

func validateVisiblePixel(t *testing.T, name string, logo pixelart.PixelArt, p pixelart.Pixel) {
	t.Helper()
	validatePixelInBounds(t, name, logo, p)
	if p.Row < 0 || p.Row >= logo.Height || p.Col < 0 || p.Col >= logo.Width {
		return
	}
	if logo.Pixels[p.Row][p.Col] == 0 {
		t.Errorf("%s animation references transparent pixel at row %d col %d", name, p.Row, p.Col)
	}
}

func validatePixelInBounds(t *testing.T, name string, logo pixelart.PixelArt, p pixelart.Pixel) {
	t.Helper()
	if p.Row < 0 || p.Row >= logo.Height || p.Col < 0 || p.Col >= logo.Width {
		t.Errorf("%s animation references out-of-bounds pixel row %d col %d", name, p.Row, p.Col)
	}
}
