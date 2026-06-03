package pkg

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FilePersister implements Persister + Watchable for file-backed configs.
type FilePersister struct {
	Path           string
	defaultContent []byte
	missingContent []byte

	watcher   *fsnotify.Watcher
	done      chan struct{}
	selfWrite int64 // unix nano timestamp of last self-write
	mu        sync.Mutex
}

// FilePersisterOption configures a FilePersister.
type FilePersisterOption func(*FilePersister)

// WithDefaultContent sets content to write if the file doesn't exist on Load.
func WithDefaultContent(data []byte) FilePersisterOption {
	return func(fp *FilePersister) {
		fp.defaultContent = data
	}
}

// WithMissingContent returns content for a missing file without creating it.
func WithMissingContent(data []byte) FilePersisterOption {
	return func(fp *FilePersister) {
		fp.missingContent = data
	}
}

// NewFilePersister creates a file-backed persister for the given path.
func NewFilePersister(path string, opts ...FilePersisterOption) *FilePersister {
	fp := &FilePersister{Path: path}
	for _, o := range opts {
		o(fp)
	}
	return fp
}

// Load reads the file from disk.
// if the file is missing and defaultContent is set, materialises it on disk first.
// if missing with no defaultContent, returns empty bytes (no error) so the konfable is
// editable as a brand-new config; the first Save will create the file on disk.
func (fp *FilePersister) Load(_ context.Context) ([]byte, error) {
	if !FileExists(fp.Path) {
		switch {
		case fp.defaultContent != nil:
			if err := EnsureDir(filepath.Dir(fp.Path)); err != nil {
				return nil, fmt.Errorf("ensure dir for %s: %w", fp.Path, err)
			}
			if err := os.WriteFile(fp.Path, fp.defaultContent, 0o644); err != nil {
				return nil, fmt.Errorf("create default %s: %w", fp.Path, err)
			}
		case fp.missingContent != nil:
			return bytes.Clone(fp.missingContent), nil
		default:
			return []byte{}, nil
		}
	}
	data, err := os.ReadFile(fp.Path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", fp.Path, err)
	}
	return data, nil
}

// Save writes a .bak backup of original, then atomically writes data.
// when the target file does not yet exist (first save of a new config), the backup
// step is skipped — there is nothing to back up and the parent dir may not exist yet.
func (fp *FilePersister) Save(_ context.Context, original, data []byte) error {
	// preserve original file permissions, fall back to 0644 for new files
	perm := os.FileMode(0o644)
	exists := false
	if info, err := os.Stat(fp.Path); err == nil {
		perm = info.Mode().Perm()
		exists = true
	}

	if err := EnsureDir(filepath.Dir(fp.Path)); err != nil {
		return fmt.Errorf("ensure dir for %s: %w", fp.Path, err)
	}
	if exists {
		bakPath := fp.Path + ".bak"
		if err := os.WriteFile(bakPath, original, perm); err != nil {
			return fmt.Errorf("backup %s: %w", bakPath, err)
		}
	}
	fp.mu.Lock()
	fp.selfWrite = time.Now().UnixNano()
	fp.mu.Unlock()
	if err := AtomicWrite(fp.Path, data, perm); err != nil {
		return fmt.Errorf("save %s: %w", fp.Path, err)
	}
	return nil
}

// Watch monitors the file for external changes via fsnotify.
// onChange is called (debounced) when an external write is detected.
func (fp *FilePersister) Watch(onChange func()) error {
	fp.mu.Lock()

	if fp.watcher != nil {
		fp.mu.Unlock()
		return nil
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		fp.mu.Unlock()
		return fmt.Errorf("create watcher: %w", err)
	}

	if err := w.Add(filepath.Dir(fp.Path)); err != nil {
		fp.mu.Unlock()
		w.Close()
		return fmt.Errorf("watch %s: %w", fp.Path, err)
	}

	fp.done = make(chan struct{})
	fp.watcher = w
	done := fp.done
	fp.mu.Unlock()

	go func() {
		var debounce *time.Timer
		const debounceMs = 100

		for {
			select {
			case <-done:
				return
			case ev, ok := <-w.Events:
				if !ok {
					return
				}

				// react to direct writes/creates and renames that replace the target
				relevant := ev.Op&(fsnotify.Write|fsnotify.Create) != 0 && ev.Name == fp.Path
				if !relevant && ev.Op&fsnotify.Rename != 0 && ev.Name == fp.Path {
					// editor did atomic rename — the old file was renamed away.
					// a Create event for the new file at our path typically follows,
					// but some editors (vim with backupcopy=no) only emit Rename.
					// check if the file exists at our path now.
					if _, err := os.Stat(fp.Path); err == nil {
						relevant = true
					}
				}
				if !relevant {
					continue
				}

				// suppress self-write events
				fp.mu.Lock()
				selfTS := fp.selfWrite
				fp.mu.Unlock()
				if time.Since(time.Unix(0, selfTS)) < 500*time.Millisecond {
					continue
				}

				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(debounceMs*time.Millisecond, onChange)

			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	return nil
}

// Unwatch closes the file watcher.
func (fp *FilePersister) Unwatch() {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if fp.done != nil {
		close(fp.done)
		fp.done = nil
	}
	if fp.watcher != nil {
		fp.watcher.Close()
		fp.watcher = nil
	}
}
