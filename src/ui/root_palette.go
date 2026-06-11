package ui

// buildPaletteItems creates the command palette entries from current state.
func (r *root) buildPaletteItems() []PaletteItem {
	var items []PaletteItem

	// app entries
	for i, k := range r.allKonfables {
		idx := i
		items = append(items, PaletteItem{
			Label:      k.Name(),
			Category:   "app",
			MatchTerms: k.Name(),
			Action:     SelectAppMsg{Index: idx},
		})
	}

	// actions
	items = append(items,
		PaletteItem{
			Label:      "Save",
			Shortcut:   "ctrl+s",
			Category:   "action",
			MatchTerms: "save write",
			Action:     SaveMsg{},
		},
		PaletteItem{
			Label:      "Undo",
			Shortcut:   "ctrl+z",
			Category:   "action",
			MatchTerms: "undo revert",
			Action:     UndoMsg{},
		},
		PaletteItem{
			Label:      "Redo",
			Shortcut:   "ctrl+y",
			Category:   "action",
			MatchTerms: "redo repeat",
			Action:     RedoMsg{},
		},
		PaletteItem{
			Label:      "Toggle Configured Filter",
			Shortcut:   "f",
			Category:   "action",
			MatchTerms: "filter configured only",
			Action:     ToggleFilterMsg{},
		},
		PaletteItem{
			Label:      "Cycle Theme",
			Shortcut:   "t",
			Category:   "action",
			MatchTerms: "theme cycle color palette",
			Action:     CycleThemeMsg{},
		},
		PaletteItem{
			Label:      "Open in $EDITOR",
			Shortcut:   "e",
			Category:   "action",
			MatchTerms: "editor vim nvim external",
			Action:     OpenEditorMsg{},
		},
		PaletteItem{
			Label:      "Help",
			Shortcut:   "?",
			Category:   "action",
			MatchTerms: "help keybindings shortcuts",
			Action:     ToggleHelpMsg{},
		},
	)

	return items
}

// buildFieldItems creates palette entries for the current app's schema fields.
func (r *root) buildFieldItems() []PaletteItem {
	if len(r.content.fields) == 0 {
		return nil
	}
	app := ""
	if r.content.konfable != nil {
		app = r.content.konfable.Name()
	}
	items := make([]PaletteItem, 0, len(r.content.fields))
	for i := range r.content.fields {
		f := &r.content.fields[i]
		section := ""
		if r.content.schema != nil && i < len(r.content.fieldSection) {
			si := r.content.fieldSection[i]
			if si < len(r.content.schema.Sections) {
				section = r.content.schema.Sections[si].Name
			}
		}
		desc := f.Type
		if f.Description != "" {
			desc = f.Description
		}
		terms := f.Key + " " + section + " " + f.Type
		if f.Description != "" {
			terms += " " + f.Description
		}
		cat := app
		if section != "" {
			cat = section
		}
		items = append(items, PaletteItem{
			Label:       f.Key,
			Description: desc,
			Category:    cat,
			MatchTerms:  terms,
			Action:      JumpToFieldMsg{FieldIdx: i},
		})
	}
	return items
}
