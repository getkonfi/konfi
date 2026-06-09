package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// logoAnimCmd schedules the next logo animation frame.
func logoAnimCmd(gen, tickMs int) tea.Cmd {
	if tickMs <= 0 {
		tickMs = 60
	}
	return tea.Tick(time.Duration(tickMs)*time.Millisecond, func(time.Time) tea.Msg {
		return logoAnimTickMsg{gen: gen}
	})
}
