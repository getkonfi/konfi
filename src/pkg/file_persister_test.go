package pkg

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFilePersisterLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")
	content := []byte("key = value\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	fp := NewFilePersister(path)
	data, err := fp.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("got %q, want %q", data, content)
	}
}

func TestFilePersisterLoadCreatesDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "new.conf")
	defaultContent := []byte("theme: dark\n")

	fp := NewFilePersister(path, WithDefaultContent(defaultContent))
	data, err := fp.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, defaultContent) {
		t.Errorf("got %q, want %q", data, defaultContent)
	}

	// file should exist on disk now
	if !FileExists(path) {
		t.Error("expected file to be created on disk")
	}
}

func TestFilePersisterLoadMissingNoDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.conf")

	fp := NewFilePersister(path)
	_, err := fp.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for missing file without default")
	}
}

func TestFilePersisterSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")
	original := []byte("original content\n")
	updated := []byte("updated content\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}

	fp := NewFilePersister(path)
	err := fp.Save(context.Background(), original, updated)
	if err != nil {
		t.Fatal(err)
	}

	// main file should have updated content
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, updated) {
		t.Errorf("main file: got %q, want %q", got, updated)
	}

	// backup should have original content
	bakData, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bakData, original) {
		t.Errorf("backup: got %q, want %q", bakData, original)
	}
}

func TestFilePersisterWatchUnwatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "watch.conf")
	if err := os.WriteFile(path, []byte("init\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fp := NewFilePersister(path)
	changed := make(chan struct{}, 1)
	err := fp.Watch(func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	// write externally
	time.Sleep(50 * time.Millisecond)
	if err := os.WriteFile(path, []byte("external\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-changed:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for change notification")
	}

	fp.Unwatch()
}

func TestFilePersisterSelfWriteSuppression(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "selfwrite.conf")
	original := []byte("before\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}

	fp := NewFilePersister(path)
	changed := make(chan struct{}, 1)
	err := fp.Watch(func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	// self-write via Save should be suppressed
	err = fp.Save(context.Background(), original, []byte("after\n"))
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-changed:
		t.Fatal("self-write should not trigger onChange")
	case <-time.After(300 * time.Millisecond):
		// ok — no notification
	}

	fp.Unwatch()
}
