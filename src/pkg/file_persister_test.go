package pkg

import (
	"bytes"
	"context"
	"fmt"
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

func TestFilePersisterLoadMissingContentDoesNotCreateFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "new.conf")
	content := []byte("seed = true\n")

	fp := NewFilePersister(path, WithMissingContent(content))
	data, err := fp.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("got %q, want %q", data, content)
	}
	if FileExists(path) {
		t.Error("missing content should not create the file on load")
	}
}

func TestFilePersisterLoadMissingNoDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.conf")

	fp := NewFilePersister(path)
	data, err := fp.Load(context.Background())
	if err != nil {
		t.Fatalf("missing file with no default should not error: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("missing file with no default should return empty bytes, got %q", data)
	}
	// the load itself must not create the file — first Save is what materializes it.
	if FileExists(path) {
		t.Error("Load must not create the file when no default is set")
	}
}

func TestFilePersisterSaveCreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "fresh.conf")
	content := []byte("key = value\n")

	fp := NewFilePersister(path)
	// load returns empty for a missing file
	original, err := fp.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(original) != 0 {
		t.Fatalf("expected empty original, got %q", original)
	}

	// first save creates the file (and its parent dir) without writing a stale .bak
	if err := fp.Save(context.Background(), original, content); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("got %q, want %q", got, content)
	}
	if FileExists(BackupPath(path)) {
		t.Error("first save should not create a backup (nothing to back up)")
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
	bakData, err := os.ReadFile(BackupPath(path))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bakData, original) {
		t.Errorf("backup: got %q, want %q", bakData, original)
	}
}

func TestFilePersisterSaveDoesNotTouchPlainBak(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")
	original := []byte("original content\n")
	updated := []byte("updated content\n")
	otherBackup := []byte("other backup\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".bak", otherBackup, 0o644); err != nil {
		t.Fatal(err)
	}

	fp := NewFilePersister(path)
	if err := fp.Save(context.Background(), original, updated); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, updated) {
		t.Errorf("main file: got %q, want %q", got, updated)
	}

	bakData, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bakData, otherBackup) {
		t.Errorf("plain backup: got %q, want %q", bakData, otherBackup)
	}

	konfiBakData, err := os.ReadFile(BackupPath(path))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(konfiBakData, original) {
		t.Errorf("konfi backup: got %q, want %q", konfiBakData, original)
	}
}

func TestFilePersisterSaveUsesNextKonfiBackupSlot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")
	original := []byte("original content\n")
	updated := []byte("updated content\n")
	existingBackup := []byte("older backup\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(BackupPath(path), existingBackup, 0o644); err != nil {
		t.Fatal(err)
	}

	fp := NewFilePersister(path)
	if err := fp.Save(context.Background(), original, updated); err != nil {
		t.Fatal(err)
	}

	bakData, err := os.ReadFile(BackupPath(path))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bakData, existingBackup) {
		t.Errorf("existing backup: got %q, want %q", bakData, existingBackup)
	}

	nextBakData, err := os.ReadFile(BackupPath(path) + ".1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(nextBakData, original) {
		t.Errorf("next backup: got %q, want %q", nextBakData, original)
	}
}

func TestFilePersisterSaveReplacesOldestBackupWhenSlotsFull(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")
	original := []byte("original content\n")
	updated := []byte("updated content\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}
	for i, bakPath := range backupPaths(path, DefaultBackupLimit) {
		if err := os.WriteFile(bakPath, []byte(fmt.Sprintf("occupied %d\n", i)), 0o644); err != nil {
			t.Fatal(err)
		}
		ts := time.Unix(int64(i+1), 0)
		if err := os.Chtimes(bakPath, ts, ts); err != nil {
			t.Fatal(err)
		}
	}

	fp := NewFilePersister(path)
	if err := fp.Save(context.Background(), original, updated); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, updated) {
		t.Errorf("main file: got %q, want %q", got, updated)
	}

	oldestData, err := os.ReadFile(BackupPath(path))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(oldestData, original) {
		t.Errorf("oldest backup: got %q, want %q", oldestData, original)
	}
	if FileExists(fmt.Sprintf("%s.%d", BackupPath(path), DefaultBackupLimit)) {
		t.Error("save should not create a sixth backup slot")
	}
}

func TestFilePersisterSaveHonorsConfiguredBackupLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")
	original := []byte("original content\n")
	updated := []byte("updated content\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}

	fp := NewFilePersister(path, WithBackupLimit(1))
	if err := fp.Save(context.Background(), original, updated); err != nil {
		t.Fatal(err)
	}
	if err := fp.Save(context.Background(), updated, []byte("third content\n")); err != nil {
		t.Fatal(err)
	}

	if FileExists(BackupPath(path) + ".1") {
		t.Error("save should not create a second backup slot when limit is 1")
	}
	bakData, err := os.ReadFile(BackupPath(path))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bakData, updated) {
		t.Errorf("backup: got %q, want %q", bakData, updated)
	}
}

func TestFilePersisterSaveSkipsBackupWhenLimitZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")
	original := []byte("original content\n")
	updated := []byte("updated content\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}

	fp := NewFilePersister(path, WithBackupLimit(0))
	if err := fp.Save(context.Background(), original, updated); err != nil {
		t.Fatal(err)
	}

	if FileExists(BackupPath(path)) {
		t.Error("backup should not be created when limit is 0")
	}
}

func TestFilePersisterSaveCreatesMissingDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "new.conf")

	fp := NewFilePersister(path, WithMissingContent([]byte("")))
	original, err := fp.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	updated := []byte("key value\n")
	if err := fp.Save(context.Background(), original, updated); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, updated) {
		t.Errorf("main file: got %q, want %q", got, updated)
	}
	if FileExists(BackupPath(path)) {
		t.Error("first save should not create a backup (nothing to back up)")
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
