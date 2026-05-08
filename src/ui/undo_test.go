package ui

import (
	"reflect"
	"testing"
)

// regression: list-field undo used to corrupt repeated keys because the
// EditOp's OldValue was display-form (", "-joined) while the apply path
// split on "\n" — collapsing every item into a single comma-laden string.
func TestSplitListValueAcceptsBothSeparators(t *testing.T) {
	want := []string{"foo", "bar", "baz"}

	cases := map[string]string{
		"newline-joined":     "foo\nbar\nbaz",
		"comma-space-joined": "foo, bar, baz",
		"with-trim":          "  foo \n  bar\nbaz  ",
		"with-empties":       "foo\n\nbar\nbaz\n",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			got := splitListValue(in)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("splitListValue(%q) = %v, want %v", in, got, want)
			}
		})
	}

	if got := splitListValue(""); got != nil {
		t.Errorf("splitListValue(\"\") = %v, want nil", got)
	}

	// when the value carries the canonical "\n", we must NOT also try
	// to re-split on ", ", or commas inside an item would split it.
	in := "ctrl+a, copy\nctrl+v, paste"
	got := splitListValue(in)
	wantPair := []string{"ctrl+a, copy", "ctrl+v, paste"}
	if !reflect.DeepEqual(got, wantPair) {
		t.Errorf("splitListValue(%q) = %v, want %v", in, got, wantPair)
	}
}

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
