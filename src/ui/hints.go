package ui

import "github.com/eminert/konfi/ui/editors"

func (r *root) updateHints() {
	// wire mode indicator and undo count into statusbar
	switch {
	case r.content.Editing():
		r.status.SetMode("EDIT")
	case r.content.searching || r.sidebar.searching:
		r.status.SetMode("SEARCH")
	default:
		r.status.SetMode("")
	}
	r.status.SetUndoCount(r.content.undoStack.Len())

	if r.content.Editing() {
		kind := editors.InteractionSingle
		if it, ok := r.content.detail.editor.(editors.Interactor); ok {
			kind = it.Interaction()
		}
		switch kind {
		case editors.InteractionList:
			r.content.hints = []keyHint{
				{"a", "add"}, {"d", "delete"},
				{"⏎", "edit"}, {"^S", "done"}, {"esc", "cancel"},
			}
		case editors.InteractionToggleMap:
			r.content.hints = []keyHint{
				{"␣", "toggle"}, {"a", "add"}, {"d", "delete"},
				{"⏎", "done"}, {"esc", "cancel"},
			}
		case editors.InteractionEnum:
			r.content.hints = []keyHint{
				{"↑↓", "select"}, {"⏎", "confirm"}, {"esc", "cancel"},
			}
		default:
			r.content.hints = []keyHint{
				{"⏎", "confirm"}, {"esc", "cancel"}, {"tab", "switch mode"},
			}
		}
		r.status.hints = r.content.hints
		return
	}

	switch r.focus {
	case paneSidebar:
		if r.sidebar.searching {
			r.content.hints = []keyHint{
				{"↑↓", "nav"},
				{"⏎", "select"},
				{"esc", "cancel"},
			}
		} else {
			hints := []keyHint{
				{"↑↓", "nav"},
				{"⏎", "open"},
				{"/", "search"},
			}
			hints = append(hints, []keyHint{
				{"→", "content"},
				{"esc", "cancel"},
				{"q", "quit"},
			}...)
			r.content.hints = hints
		}
	case paneContent:
		if r.content.detailFocused {
			hints := []keyHint{
				{"↑↓", "scroll file"},
				{"Pg", "page"},
				{"Home", "top"},
				{"End", "bottom"},
				{"←", "fields"},
			}
			if r.content.config != nil && r.content.config.Path != "" {
				hints = append(hints, keyHint{"e", "$EDITOR"})
			}
			hints = append(hints, keyHint{"q", "quit"})
			r.content.hints = hints
			r.status.hints = r.content.hints
			return
		}
		if r.content.searching {
			r.content.hints = []keyHint{
				{"↑↓", "nav"},
				{"⏎", "lock"},
				{"esc", "cancel"},
			}
			r.status.hints = r.content.hints
			return
		}
		r.content.hints = []keyHint{
			{"↑↓", "nav"},
			{"⏎", "edit"},
			{"⌫", "del"},
			{"/", "search"},
			{"c", "copy"},
			{"p", "paste"},
			{".", "configured"},
			{"⇥", "changed"},
			{"q", "quit"},
			{"esc", "cancel"},
		}
	}
	r.status.hints = r.content.hints
}
