package ui

import "testing"

func TestPushUndoRedo(t *testing.T) {
	s := NewUndoStack(50)

	s.Push(EditOp{FieldKey: "font-size", OldValue: "12", NewValue: "14"})
	s.Push(EditOp{FieldKey: "font-family", OldValue: "mono", NewValue: "iosevka"})

	if s.Len() != 2 {
		t.Fatalf("expected len 2, got %d", s.Len())
	}

	op, ok := s.Undo()
	if !ok || op.FieldKey != "font-family" || op.OldValue != "mono" {
		t.Fatalf("unexpected undo result: %+v, ok=%v", op, ok)
	}

	op, ok = s.Redo()
	if !ok || op.FieldKey != "font-family" || op.NewValue != "iosevka" {
		t.Fatalf("unexpected redo result: %+v, ok=%v", op, ok)
	}
}

func TestRedoClearedOnPush(t *testing.T) {
	s := NewUndoStack(50)

	s.Push(EditOp{FieldKey: "a", OldValue: "1", NewValue: "2"})
	s.Push(EditOp{FieldKey: "b", OldValue: "3", NewValue: "4"})
	s.Undo()

	if !s.CanRedo() {
		t.Fatal("expected CanRedo after undo")
	}

	// new push should clear redo
	s.Push(EditOp{FieldKey: "c", OldValue: "5", NewValue: "6"})

	if s.CanRedo() {
		t.Fatal("redo should be cleared after new push")
	}
}

func TestStackLimit(t *testing.T) {
	s := NewUndoStack(3)

	for i := 0; i < 5; i++ {
		s.Push(EditOp{FieldKey: "k", OldValue: "old", NewValue: "new"})
	}

	if s.Len() != 3 {
		t.Fatalf("expected len capped at 3, got %d", s.Len())
	}
}

func TestClear(t *testing.T) {
	s := NewUndoStack(50)

	s.Push(EditOp{FieldKey: "x", OldValue: "a", NewValue: "b"})
	s.Undo()
	s.Clear()

	if s.CanUndo() || s.CanRedo() {
		t.Fatal("expected both stacks empty after clear")
	}
}

func TestCanUndoCanRedo(t *testing.T) {
	s := NewUndoStack(50)

	if s.CanUndo() || s.CanRedo() {
		t.Fatal("empty stack should report false for both")
	}

	s.Push(EditOp{FieldKey: "k", OldValue: "a", NewValue: "b"})

	if !s.CanUndo() {
		t.Fatal("expected CanUndo after push")
	}
	if s.CanRedo() {
		t.Fatal("expected no CanRedo before undo")
	}

	s.Undo()

	if s.CanUndo() {
		t.Fatal("expected no CanUndo after undoing only op")
	}
	if !s.CanRedo() {
		t.Fatal("expected CanRedo after undo")
	}
}

func TestUndoOnEmpty(t *testing.T) {
	s := NewUndoStack(50)
	_, ok := s.Undo()
	if ok {
		t.Fatal("undo on empty stack should return false")
	}
}

func TestRedoOnEmpty(t *testing.T) {
	s := NewUndoStack(50)
	_, ok := s.Redo()
	if ok {
		t.Fatal("redo on empty stack should return false")
	}
}

func TestDefaultLimit(t *testing.T) {
	s := NewUndoStack(0)
	if s.limit != 50 {
		t.Fatalf("expected default limit 50, got %d", s.limit)
	}

	s2 := NewUndoStack(-1)
	if s2.limit != 50 {
		t.Fatalf("expected default limit 50 for negative, got %d", s2.limit)
	}
}
