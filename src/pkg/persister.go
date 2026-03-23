package pkg

import "context"

// Persister abstracts reading/writing config data from any backing store.
// file-backed apps use FilePersister; non-file backends (e.g. gsettings) provide their own.
type Persister interface {
	Load(ctx context.Context) ([]byte, error)
	Save(ctx context.Context, original, data []byte) error
}

// Watchable is an optional interface for persisters that support
// monitoring the backing store for external changes.
type Watchable interface {
	Watch(onChange func()) error
	Unwatch()
}

// TierReporter is an optional interface for multi-tier persisters.
// it exposes which config tier (e.g. "global", "local", "project") owns a key.
type TierReporter interface {
	TierOf(key string) string   // highest-precedence tier owning this key
	Tiers(key string) []string  // all tiers that define this key, highest first
}
