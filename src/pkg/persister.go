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
