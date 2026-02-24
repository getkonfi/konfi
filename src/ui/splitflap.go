package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	splitFlapTickRate = 45 * time.Millisecond
	splitFlapMaxSteps = 14
	splitFlapCharSet  = " ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789./-~:_"
)

// splitFlapState drives a Solari-board text transition.
// operates on plain strings — styling applied after frame generation.
type splitFlapState struct {
	source   []string
	target   []string
	current  []string
	step     int
	done     bool
	gen      int
	maxChars int
}

func newSplitFlap(source, target []string, gen int) *splitFlapState {
	// pad to equal number of lines
	maxLines := len(source)
	if len(target) > maxLines {
		maxLines = len(target)
	}
	src := make([]string, maxLines)
	tgt := make([]string, maxLines)
	cur := make([]string, maxLines)
	copy(src, source)
	copy(tgt, target)
	copy(cur, source)

	// find longest line to set proper animation duration
	maxChars := 0
	for _, s := range tgt {
		if n := len([]rune(s)); n > maxChars {
			maxChars = n
		}
	}

	return &splitFlapState{
		source:   src,
		target:   tgt,
		current:  cur,
		step:     0,
		done:     false,
		gen:      gen,
		maxChars: maxChars,
	}
}

// tick advances one frame. returns true when animation is complete.
func (s *splitFlapState) tick() bool {
	if s.done {
		return true
	}
	s.step++
	allDone := true
	for i := range s.current {
		s.current[i] = advanceLine(s.source[i], s.target[i], s.step, i)
		if s.current[i] != s.target[i] {
			allDone = false
		}
	}
	if allDone || s.step >= splitFlapMaxSteps+s.maxChars+len(s.current) {
		s.done = true
		copy(s.current, s.target)
	}
	return s.done
}

// advanceLine settles characters left-to-right with a 1-step stagger per line.
// positions beyond the target length settle to space immediately (no cycling).
func advanceLine(src, tgt string, step, lineOffset int) string {
	effectiveStep := step - lineOffset
	if effectiveStep <= 0 {
		return src
	}

	srcRunes := []rune(src)
	tgtRunes := []rune(tgt)
	tgtLen := len(tgtRunes)

	// pad to equal length
	maxLen := len(srcRunes)
	if tgtLen > maxLen {
		maxLen = tgtLen
	}
	for len(srcRunes) < maxLen {
		srcRunes = append(srcRunes, ' ')
	}
	for len(tgtRunes) < maxLen {
		tgtRunes = append(tgtRunes, ' ')
	}

	out := make([]rune, maxLen)
	for i := range out {
		charStep := effectiveStep - i
		switch {
		case charStep <= 0:
			out[i] = srcRunes[i]
		case i >= tgtLen:
			// beyond target text: blank immediately, no cycling
			out[i] = ' '
		case charStep >= splitFlapMaxSteps || srcRunes[i] == tgtRunes[i]:
			out[i] = tgtRunes[i]
		default:
			idx := strings.IndexRune(splitFlapCharSet, tgtRunes[i])
			if idx < 0 {
				out[i] = tgtRunes[i]
			} else {
				flipIdx := (idx + splitFlapMaxSteps - charStep) % len([]rune(splitFlapCharSet))
				out[i] = []rune(splitFlapCharSet)[flipIdx]
			}
		}
	}
	return strings.TrimRight(string(out), " ")
}

func splitFlapCmd(gen int) tea.Cmd {
	return tea.Tick(splitFlapTickRate, func(time.Time) tea.Msg {
		return splitFlapTickMsg{gen: gen}
	})
}
