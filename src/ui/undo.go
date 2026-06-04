package ui

import "strings"

// splitListValue parses a list-field value into items, accepting either the
// canonical "\n"-joined form (produced by listEditor.Value and used in the
// editor pipeline) or the display ", "-joined form stored in c.values.
//
// EditOp.OldValue is captured from c.values for some write paths (delete,
// revert, paste) and from the editor for others (commit), so the apply path
// must accept both. choosing one canonical form universally would require
// also recanonicalising c.values for list fields, which would ripple into
// every renderer that reads it — out of scope for this fix.
//
// items are trimmed and empties dropped, matching listEditor.Init's behavior.
func splitListValue(s string) []string {
	if s == "" {
		return nil
	}
	var raw []string
	if strings.Contains(s, "\n") {
		raw = strings.Split(s, "\n")
	} else {
		raw = strings.Split(s, ", ")
	}
	out := raw[:0]
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

// EditOp represents a single field edit that can be undone/redone.
type EditOp struct {
	FieldKey string
	OldValue string
	NewValue string
}

// UndoStack manages per-app undo/redo history.
// not thread-safe — TUI is single-threaded.
type UndoStack struct {
	undos []EditOp
	redos []EditOp
	limit int
}

// NewUndoStack creates a stack with the given max depth.
func NewUndoStack(limit int) *UndoStack {
	if limit <= 0 {
		limit = 50
	}
	return &UndoStack{
		undos: make([]EditOp, 0, limit),
		redos: make([]EditOp, 0, limit),
		limit: limit,
	}
}

// Push records a new edit, clearing the redo stack.
func (s *UndoStack) Push(op EditOp) {
	s.redos = s.redos[:0]
	if len(s.undos) >= s.limit {
		// drop oldest
		copy(s.undos, s.undos[1:])
		s.undos = s.undos[:len(s.undos)-1]
	}
	s.undos = append(s.undos, op)
}

// Undo pops the last edit. Caller applies OldValue.
func (s *UndoStack) Undo() (EditOp, bool) {
	if len(s.undos) == 0 {
		return EditOp{}, false
	}
	op := s.undos[len(s.undos)-1]
	s.undos = s.undos[:len(s.undos)-1]
	s.redos = append(s.redos, op)
	return op, true
}

// Redo pops from redo stack. Caller applies NewValue.
func (s *UndoStack) Redo() (EditOp, bool) {
	if len(s.redos) == 0 {
		return EditOp{}, false
	}
	op := s.redos[len(s.redos)-1]
	s.redos = s.redos[:len(s.redos)-1]
	s.undos = append(s.undos, op)
	return op, true
}

// CanUndo returns whether there are operations to undo.
func (s *UndoStack) CanUndo() bool { return len(s.undos) > 0 }

// CanRedo returns whether there are operations to redo.
func (s *UndoStack) CanRedo() bool { return len(s.redos) > 0 }

// Clear resets both stacks.
func (s *UndoStack) Clear() {
	s.undos = s.undos[:0]
	s.redos = s.redos[:0]
}

// Len returns the number of undoable operations.
func (s *UndoStack) Len() int { return len(s.undos) }

// Clone returns an independent copy of the undo/redo history.
func (s *UndoStack) Clone() *UndoStack {
	if s == nil {
		return NewUndoStack(50)
	}
	clone := NewUndoStack(s.limit)
	clone.undos = append(clone.undos, s.undos...)
	clone.redos = append(clone.redos, s.redos...)
	return clone
}
