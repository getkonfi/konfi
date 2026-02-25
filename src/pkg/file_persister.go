package pkg

import (
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

	watcher   *fsnotify.Watcher
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

// NewFilePersister creates a file-backed persister for the given path.
func NewFilePersister(path string, opts ...FilePersisterOption) *FilePersister {
	fp := &FilePersister{Path: path}
	for _, o := range opts {
		o(fp)
	}
	return fp
}

// Load reads the file from disk. if missing and defaultContent is set, creates it first.
func (fp *FilePersister) Load(_ context.Context) ([]byte, error) {
	if !FileExists(fp.Path) && fp.defaultContent != nil {
		if err := EnsureDir(filepath.Dir(fp.Path)); err != nil {
			return nil, fmt.Errorf("ensure dir for %s: %w", fp.Path, err)
		}
		if err := os.WriteFile(fp.Path, fp.defaultContent, 0o644); err != nil {
			return nil, fmt.Errorf("create default %s: %w", fp.Path, err)
		}
	}
	data, err := os.ReadFile(fp.Path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", fp.Path, err)
	}
	return data, nil
}

// Save writes a .bak backup of original, then atomically writes data.
func (fp *FilePersister) Save(_ context.Context, original, data []byte) error {
	bakPath := fp.Path + ".bak"
	if err := os.WriteFile(bakPath, original, 0o644); err != nil {
		return fmt.Errorf("backup %s: %w", bakPath, err)
	}
	if err := AtomicWrite(fp.Path, data, 0o644); err != nil {
		return fmt.Errorf("save %s: %w", fp.Path, err)
	}
	fp.mu.Lock()
	fp.selfWrite = time.Now().UnixNano()
	fp.mu.Unlock()
	return nil
}

// Watch monitors the file for external changes via fsnotify.
// onChange is called (debounced) when an external write is detected.
func (fp *FilePersister) Watch(onChange func()) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if fp.watcher != nil {
		return nil
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}

	if err := w.Add(filepath.Dir(fp.Path)); err != nil {
		w.Close()
		return fmt.Errorf("watch %s: %w", fp.Path, err)
	}

	fp.watcher = w

	go func() {
		var debounce *time.Timer
		const debounceMs = 100

		for {
			select {
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if ev.Op&(fsnotify.Write|fsnotify.Create) == 0 {
					continue
				}
				if ev.Name != fp.Path {
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

	if fp.watcher != nil {
		fp.watcher.Close()
		fp.watcher = nil
	}
}
