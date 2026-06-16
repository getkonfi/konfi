package pkg

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/getkonfi/konfi/pkg/parser"

	"golang.org/x/sync/errgroup"
)

// CommandPersister implements Persister for backends fronted by external
// commands (e.g. dconf, gsettings) rather than a config file. Load reads each
// managed key concurrently and emits synthetic "key = value\n" lines; Save
// diffs against the original and writes only changed keys. it intentionally
// does not implement Watchable — there's no cheap way to watch these stores.
type CommandPersister[K any] struct {
	// Keys are the managed keys to load.
	Keys []K
	// LineKey maps a key to the stable identifier used in the flat output.
	LineKey func(K) string
	// Read returns the current value for a key; a non-nil error skips it.
	Read func(ctx context.Context, k K) (string, error)
	// Write applies a changed value, addressed by its LineKey.
	Write func(ctx context.Context, lineKey, value string) error
	// Delete removes or resets a key addressed by its LineKey.
	Delete func(ctx context.Context, lineKey string) error
	// ErrPrefix labels aggregated save errors (e.g. "dconf write").
	ErrPrefix string
}

func (c *CommandPersister[K]) Load(ctx context.Context) ([]byte, error) {
	vals := make([]string, len(c.Keys))
	ok := make([]bool, len(c.Keys))

	g, gctx := errgroup.WithContext(ctx)
	for i, k := range c.Keys {
		g.Go(func() error {
			v, err := c.Read(gctx, k)
			if err != nil {
				return nil // skip keys absent on this system
			}
			vals[i], ok[i] = v, true
			return nil
		})
	}
	_ = g.Wait() // per-key errors are swallowed above

	var buf bytes.Buffer
	for i, k := range c.Keys {
		if ok[i] {
			fmt.Fprintf(&buf, "%s = %s\n", c.LineKey(k), vals[i])
		}
	}
	return buf.Bytes(), nil
}

func (c *CommandPersister[K]) Save(ctx context.Context, original, data []byte) error {
	flat := parser.FlatParser{Split: parser.SplitSpacedEquals}
	origMap := flat.FindAll(original)
	newMap := flat.FindAll(data)

	var errs []string
	for key := range origMap {
		if _, ok := newMap[key]; ok {
			continue
		}
		if c.Delete == nil {
			errs = append(errs, fmt.Sprintf("%s: delete unsupported", key))
			continue
		}
		if err := c.Delete(ctx, key); err != nil {
			errs = append(errs, fmt.Sprintf("%s: delete: %v", key, err))
		}
	}
	for key, newVal := range newMap {
		if origVal, ok := origMap[key]; ok && origVal == newVal {
			continue
		}
		if err := c.Write(ctx, key, newVal); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", key, err))
		}
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("%s failed: %s", c.ErrPrefix, strings.Join(errs, "; "))
	}
	return nil
}
