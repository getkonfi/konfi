package pkg

import (
	"bytes"
	"context"
	"fmt"
	"sync"
)

// ConfigFile manages loaded config data with dirty tracking.
// I/O is delegated to a Persister — ConfigFile is backend-agnostic.
type ConfigFile struct {
	Path      string
	persister Persister
	original  []byte
	current   []byte
	dirty     bool
	gen       uint64
	mu        sync.Mutex
}

// NewConfigFile creates a ConfigFile by loading data through the given persister.
func NewConfigFile(ctx context.Context, p Persister) (*ConfigFile, error) {
	data, err := p.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	cf := &ConfigFile{
		persister: p,
		original:  data,
		current:   bytes.Clone(data),
	}

	// set Path from FilePersister if applicable
	if fp, ok := p.(*FilePersister); ok {
		cf.Path = fp.Path
	}

	return cf, nil
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
	cf.gen++
}

// Generation returns a monotonically increasing counter bumped on every content change.
func (cf *ConfigFile) Generation() uint64 {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	return cf.gen
}

// Dirty returns whether the working copy differs from the original.
func (cf *ConfigFile) Dirty() bool {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	return cf.dirty
}

// Save persists current data through the persister and clears dirty state.
func (cf *ConfigFile) Save(ctx context.Context) error {
	cf.mu.Lock()
	original := bytes.Clone(cf.original)
	current := bytes.Clone(cf.current)
	cf.mu.Unlock()

	if err := cf.persister.Save(ctx, original, current); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	cf.mu.Lock()
	cf.original = current
	cf.dirty = false
	cf.mu.Unlock()
	return nil
}

// TierOf returns which tier a key comes from, or "" if the persister doesn't support tiers.
func (cf *ConfigFile) TierOf(key string) string {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	if tr, ok := cf.persister.(TierReporter); ok {
		return tr.TierOf(key)
	}
	return ""
}

// Tiers returns all tiers that define a key, or nil if the persister doesn't support tiers.
func (cf *ConfigFile) Tiers(key string) []string {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	if tr, ok := cf.persister.(TierReporter); ok {
		return tr.Tiers(key)
	}
	return nil
}

// Preview writes the working copy to disk without promoting it to original.
// dirty state is preserved so the user can still save or revert.
func (cf *ConfigFile) Preview(ctx context.Context) error {
	cf.mu.Lock()
	original := bytes.Clone(cf.original)
	current := bytes.Clone(cf.current)
	cf.mu.Unlock()

	if err := cf.persister.Save(ctx, original, current); err != nil {
		return fmt.Errorf("preview config: %w", err)
	}
	// intentionally do NOT update cf.original — dirty stays true
	return nil
}

// RevertPreview restores the original content to disk, undoing a preview.
func (cf *ConfigFile) RevertPreview(ctx context.Context) error {
	cf.mu.Lock()
	original := bytes.Clone(cf.original)
	current := bytes.Clone(cf.current)
	cf.mu.Unlock()

	if err := cf.persister.Save(ctx, current, original); err != nil {
		return fmt.Errorf("revert preview: %w", err)
	}
	return nil
}

// Reload re-reads data from the persister, resetting both original and current.
func (cf *ConfigFile) Reload(ctx context.Context) error {
	data, err := cf.persister.Load(ctx)
	if err != nil {
		return fmt.Errorf("reload config: %w", err)
	}

	cf.mu.Lock()
	cf.original = data
	cf.current = bytes.Clone(data)
	cf.dirty = false
	cf.gen++
	cf.mu.Unlock()
	return nil
}
