package ui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

const (
	splitFlapTickRate         = 45 * time.Millisecond
	splitFlapMaxSteps         = 14
	splitFlapColumnStaggerNum = 4
	splitFlapColumnStaggerDen = 5
	splitFlapCharSet          = " ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789./-~:_"
)

var splitFlapRunes = []rune(splitFlapCharSet)

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

	return &splitFlapState{
		source:   src,
		target:   tgt,
		current:  cur,
		step:     0,
		done:     false,
		gen:      gen,
		maxChars: maxLineRunes(tgt),
	}
}

func (s *splitFlapState) retarget(target []string) {
	maxLines := len(s.current)
	if len(target) > maxLines {
		maxLines = len(target)
	}
	for len(s.source) < maxLines {
		s.source = append(s.source, "")
	}
	for len(s.current) < maxLines {
		s.current = append(s.current, "")
	}
	tgt := make([]string, maxLines)
	copy(tgt, target)
	s.target = tgt
	s.maxChars = maxLineRunes(tgt)
}

func maxLineRunes(lines []string) int {
	maxChars := 0
	for _, line := range lines {
		if n := len([]rune(line)); n > maxChars {
			maxChars = n
		}
	}
	return maxChars
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

// advanceLine settles characters left-to-right with a compressed column stagger.
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
		charStep := effectiveStep - splitFlapColumnDelay(i)
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
				flipIdx := (idx + splitFlapMaxSteps - charStep) % len(splitFlapRunes)
				out[i] = splitFlapRunes[flipIdx]
			}
		}
	}
	return strings.TrimRight(string(out), " ")
}

func splitFlapColumnDelay(col int) int {
	if col <= 0 {
		return 0
	}
	return (col*splitFlapColumnStaggerNum + splitFlapColumnStaggerDen/2) / splitFlapColumnStaggerDen
}

func splitFlapCmd(gen int) tea.Cmd {
	return tea.Tick(splitFlapTickRate, func(time.Time) tea.Msg {
		return splitFlapTickMsg{gen: gen}
	})
}
