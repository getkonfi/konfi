package pkg

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestCommandPersisterLoad(t *testing.T) {
	cp := &CommandPersister[string]{
		Keys:    []string{"a", "missing", "b"},
		LineKey: func(k string) string { return k },
		Read: func(_ context.Context, k string) (string, error) {
			if k == "missing" {
				return "", errors.New("absent")
			}
			return "val-" + k, nil
		},
		Write:     func(context.Context, string, string) error { return nil },
		ErrPrefix: "test write",
	}

	out, err := cp.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// errored keys are skipped; output preserves key order.
	want := "a = val-a\nb = val-b\n"
	if string(out) != want {
		t.Errorf("Load = %q, want %q", out, want)
	}
}

func TestCommandPersisterSaveOnlyChanged(t *testing.T) {
	var wrote []string
	cp := &CommandPersister[string]{
		LineKey: func(k string) string { return k },
		Write: func(_ context.Context, lineKey, value string) error {
			wrote = append(wrote, lineKey+"="+value)
			return nil
		},
		ErrPrefix: "test write",
	}

	original := []byte("a = 1\nb = 2\nc = 3\n")
	data := []byte("a = 1\nb = 20\nc = 3\nd = 4\n")
	if err := cp.Save(context.Background(), original, data); err != nil {
		t.Fatalf("Save: %v", err)
	}

	sort.Strings(wrote)
	want := []string{"b=20", "d=4"} // unchanged a/c skipped; changed b and new d written
	if len(wrote) != len(want) {
		t.Fatalf("wrote %v, want %v", wrote, want)
	}
	for i := range want {
		if wrote[i] != want[i] {
			t.Errorf("wrote[%d] = %q, want %q", i, wrote[i], want[i])
		}
	}
}

func TestCommandPersisterSaveAggregatesErrors(t *testing.T) {
	cp := &CommandPersister[string]{
		LineKey:   func(k string) string { return k },
		Write:     func(context.Context, string, string) error { return fmt.Errorf("boom") },
		ErrPrefix: "dconf write",
	}

	err := cp.Save(context.Background(), nil, []byte("z = 1\na = 2\n"))
	if err == nil {
		t.Fatal("expected aggregated error, got nil")
	}
	msg := err.Error()
	if !strings.HasPrefix(msg, "dconf write failed: ") {
		t.Errorf("missing prefix: %q", msg)
	}
	// errors are sorted by key, so "a" precedes "z" regardless of map order.
	if ai, zi := strings.Index(msg, "a: boom"), strings.Index(msg, "z: boom"); ai < 0 || zi < 0 || ai > zi {
		t.Errorf("errors not sorted: %q", msg)
	}
}
