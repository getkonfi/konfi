package ui

import (
	"strings"
	"testing"
)

func TestSplitFlapColumnStaggerIsCompressed(t *testing.T) {
	const col = 24

	delay := splitFlapColumnDelay(col)
	reduction := float64(col-delay) / float64(col)
	if reduction < 0.18 || reduction > 0.23 {
		t.Fatalf("column delay reduction = %.3f, want 18%%..23%%", reduction)
	}
}

func TestAdvanceLineUsesCompressedColumnStagger(t *testing.T) {
	target := strings.Repeat("A", 25)
	settledStep := splitFlapMaxSteps + splitFlapColumnDelay(len(target)-1)

	if got := advanceLine("", target, settledStep, 0); got != target {
		t.Fatalf("compressed stagger did not settle target at step %d: %q", settledStep, got)
	}
	if got := advanceLine("", target, settledStep-1, 0); got == target {
		t.Fatalf("compressed stagger settled target too early at step %d", settledStep-1)
	}
}
