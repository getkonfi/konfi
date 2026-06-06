package ui

import tea "charm.land/bubbletea/v2"

// navEntry records a navigation position for back/forward.
type navEntry struct {
	appIndex int
}

// pushNav records the current position in the navigation history.
func (r *root) pushNav() {
	entry := navEntry{
		appIndex: r.sidebar.appIndex(),
	}
	// truncate forward history when pushing
	if r.navHistoryPos < len(r.navHistory) {
		r.navHistory = r.navHistory[:r.navHistoryPos]
	}
	r.navHistory = append(r.navHistory, entry)
	r.navHistoryPos = len(r.navHistory)
}

// navBack navigates to the previous position in history.
func (r *root) navBack() tea.Cmd {
	if r.navHistoryPos <= 0 || len(r.navHistory) == 0 {
		return nil
	}
	// save current position at the tip so forward can return here
	if r.navHistoryPos >= len(r.navHistory) {
		r.navHistory = append(r.navHistory, navEntry{appIndex: r.sidebar.appIndex()})
	}
	r.navHistoryPos--
	entry := r.navHistory[r.navHistoryPos]
	r.sidebar.setCursorToApp(entry.appIndex)
	idx := entry.appIndex
	return func() tea.Msg {
		return AppSelectedMsg{Index: idx, Confirmed: true}
	}
}

// navForward navigates to the next position in history.
func (r *root) navForward() tea.Cmd {
	if r.navHistoryPos >= len(r.navHistory)-1 {
		return nil
	}
	r.navHistoryPos++
	entry := r.navHistory[r.navHistoryPos]
	r.sidebar.setCursorToApp(entry.appIndex)
	idx := entry.appIndex
	return func() tea.Msg {
		return AppSelectedMsg{Index: idx, Confirmed: true}
	}
}
