package pkg

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ConfigFile manages a loaded config file with dirty tracking and file watching.
type ConfigFile struct {
	Path      string
	original  []byte
	current   []byte
	dirty     bool
	watcher   *fsnotify.Watcher
	selfWrite int64 // unix nano timestamp of last self-write
	mu        sync.Mutex
}

// LoadOrCreateConfigFile loads the config at path, or creates it with
// defaultContent if it doesn't exist.
func LoadOrCreateConfigFile(path string, defaultContent []byte) (*ConfigFile, error) {
	if !FileExists(path) {
		if err := EnsureDir(filepath.Dir(path)); err != nil {
			return nil, fmt.Errorf("ensure dir for %s: %w", path, err)
		}
		if err := os.WriteFile(path, defaultContent, 0o644); err != nil {
			return nil, fmt.Errorf("create default %s: %w", path, err)
		}
	}
	return LoadConfigFile(path)
}

// LoadConfigFile reads a config file from disk.
func LoadConfigFile(path string) (*ConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	return &ConfigFile{
		Path:     path,
		original: data,
		current:  bytes.Clone(data),
	}, nil
}

// Content returns a copy of the current working copy.
func (cf *ConfigFile) Content() []byte {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	return bytes.Clone(cf.current)
}

// SetContent replaces the working copy and recomputes dirty state.
func (cf *ConfigFile) SetContent(data []byte) {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	cf.current = bytes.Clone(data)
	cf.dirty = !bytes.Equal(cf.current, cf.original)
}

// Dirty returns whether the working copy differs from the original.
func (cf *ConfigFile) Dirty() bool {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	return cf.dirty
}

// Save backs up the original, atomically writes current, and clears dirty.
func (cf *ConfigFile) Save() error {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	// backup original
	bakPath := cf.Path + ".bak"
	if err := os.WriteFile(bakPath, cf.original, 0o644); err != nil {
		return fmt.Errorf("backup %s: %w", bakPath, err)
	}

	if err := AtomicWrite(cf.Path, cf.current, 0o644); err != nil {
		return fmt.Errorf("save %s: %w", cf.Path, err)
	}

	cf.selfWrite = time.Now().UnixNano()
	cf.original = bytes.Clone(cf.current)
	cf.dirty = false
	return nil
}

// Reload re-reads the file from disk, resetting both original and current.
func (cf *ConfigFile) Reload() error {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	data, err := os.ReadFile(cf.Path)
	if err != nil {
		return fmt.Errorf("reload %s: %w", cf.Path, err)
	}

	cf.original = data
	cf.current = bytes.Clone(data)
	cf.dirty = false
	return nil
}

// StartWatching monitors the file for external changes.
// onChange is called (debounced) when an external write is detected.
func (cf *ConfigFile) StartWatching(onChange func()) error {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	if cf.watcher != nil {
		return nil
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}

	if err := w.Add(filepath.Dir(cf.Path)); err != nil {
		w.Close()
		return fmt.Errorf("watch %s: %w", cf.Path, err)
	}

	cf.watcher = w

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
				// only react to our specific file
				if ev.Name != cf.Path {
					continue
				}

				// suppress self-write events
				cf.mu.Lock()
				selfTS := cf.selfWrite
				cf.mu.Unlock()
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

// StopWatching closes the file watcher.
func (cf *ConfigFile) StopWatching() {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	if cf.watcher != nil {
		cf.watcher.Close()
		cf.watcher = nil
	}
}
