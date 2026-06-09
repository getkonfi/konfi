package pixelart

import "testing"

func TestSequenceAnimationAppliesActiveGroupAndTrail(t *testing.T) {
	base := PixelArt{
		Width: 4, Height: 2,
		Pixels: [][]uint8{
			{1, 1, 1, 1},
			{1, 1, 1, 1},
		},
	}
	state := NewAnimState(base, AnimConfig{
		Kind:           AnimSequence,
		Frames:         2,
		TickMs:         1,
		SequenceGroups: [][]Pixel{{{Row: 0, Col: 0}}, {{Row: 0, Col: 1}}},
		SequenceSeq:    []int{-1, 1},
		SequenceBright: []uint8{9, 8},
	})

	frame := state.CurrentFrame()
	if got := frame.Pixels[0][0]; got != 1 {
		t.Fatalf("inactive frame changed pixel = %d, want 1", got)
	}

	state.Tick()
	frame = state.CurrentFrame()
	if got := frame.Pixels[0][1]; got != 9 {
		t.Fatalf("active group color = %d, want 9", got)
	}
	if got := frame.Pixels[0][0]; got != 8 {
		t.Fatalf("trail group color = %d, want 8", got)
	}
}
