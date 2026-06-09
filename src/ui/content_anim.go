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
