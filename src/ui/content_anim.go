package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// logoAnimCmd schedules the next logo animation frame at 60ms.
func logoAnimCmd(gen int) tea.Cmd {
	return tea.Tick(60*time.Millisecond, func(time.Time) tea.Msg {
		return logoAnimTickMsg{gen: gen}
	})
}

// insightTickCmd starts the next insight cycle timer.
func (c content) insightTickCmd() tea.Cmd {
	gen := c.insightGen
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		return insightTickMsg{gen: gen}
	})
}
