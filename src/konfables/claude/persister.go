package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/emin/konfigurator/pkg"
	"github.com/fsnotify/fsnotify"
)

// tier names in precedence order (lowest to highest).
const (
	TierGlobal  = "global"
	TierLocal   = "local"
	TierProject = "project"
)

// tierDef pairs a tier name with its file path.
type tierDef struct {
	Name string
	Path string
}

// TieredPersister implements pkg.Persister, pkg.Watchable, and pkg.TierReporter
// for Claude Code's 3-tier JSON settings: global < local < project.
type TieredPersister struct {
	tiers []tierDef // ordered lowest → highest precedence

	mu        sync.Mutex
	tierMap   map[string][]string // dottedKey → tier names (highest first)
	watcher   *fsnotify.Watcher
	selfWrite int64 // unix-nano of last self-initiated write
}

// NewTieredPersister creates a persister for the given tier paths.
// paths are ordered: global, local, project (lowest → highest precedence).
func NewTieredPersister(globalPath, localPath, projectPath string) *TieredPersister {
	return &TieredPersister{
		tiers: []tierDef{
			{Name: TierGlobal, Path: globalPath},
			{Name: TierLocal, Path: localPath},
			{Name: TierProject, Path: projectPath},
		},
		tierMap: make(map[string][]string),
	}
}

// Load reads all tier files, deep-merges them, and builds the tier provenance map.
func (tp *TieredPersister) Load(_ context.Context) ([]byte, error) {
	merged, tierMap, err := tp.mergeAll()
	if err != nil {
		return nil, err
	}

	tp.mu.Lock()
	tp.tierMap = tierMap
	tp.mu.Unlock()

	return marshalJSON(merged)
}

// Save diffs original (merged blob) vs data (new merged blob), determines which
// tier files need updating, and writes only those files.
func (tp *TieredPersister) Save(_ context.Context, original, data []byte) error {
	origKeys := listAllKeys(original)
	newKeys := listAllKeys(data)

	tp.mu.Lock()
	tierMap := copyTierMap(tp.tierMap)
	tp.mu.Unlock()

	// load each tier's current content
	tierData := make(map[string][]byte, len(tp.tiers))
	for _, t := range tp.tiers {
		raw, err := readFileOrEmpty(t.Path)
		if err != nil {
			return fmt.Errorf("read tier %s: %w", t.Name, err)
		}
		tierData[t.Name] = raw
	}

	parser := &pkg.JSONParser{}
	dirty := make(map[string]bool)

	// handle changed/added keys
	for _, key := range newKeys {
		oldVal, hadOld := parser.FindValue(original, key)
		newVal, _ := parser.FindValue(data, key)
		if hadOld && oldVal == newVal {
			continue
		}

		tier := tp.routeKey(key, tierMap)
		td := tierData[tier]
		updated, err := parser.SetValue(td, key, newVal)
		if err != nil {
			return fmt.Errorf("set %s in tier %s: %w", key, tier, err)
		}
		tierData[tier] = updated
		dirty[tier] = true
	}

	// handle deleted keys
	for _, key := range origKeys {
		if containsKey(newKeys, key) {
			continue
		}
		tier := tp.ownerTier(key, tierMap)
		if tier == "" {
			continue
		}
		td := tierData[tier]
		updated, err := parser.DeleteKey(td, key)
		if err != nil {
			// key might not exist in this specific tier file, skip
			continue
		}
		tierData[tier] = updated
		dirty[tier] = true
	}

	// write dirty tiers
	for _, t := range tp.tiers {
		if !dirty[t.Name] {
			continue
		}
		if err := pkg.EnsureDir(filepath.Dir(t.Path)); err != nil {
			return fmt.Errorf("ensure dir for %s: %w", t.Name, err)
		}
		if err := pkg.AtomicWrite(t.Path, tierData[t.Name], 0o644); err != nil {
			return fmt.Errorf("save tier %s: %w", t.Name, err)
		}
	}

	if len(dirty) > 0 {
		tp.mu.Lock()
		tp.selfWrite = time.Now().UnixNano()
		tp.mu.Unlock()
	}

	// rebuild tier map after save
	_, newTierMap, err := tp.mergeAll()
	if err == nil {
		tp.mu.Lock()
		tp.tierMap = newTierMap
		tp.mu.Unlock()
	}

	return nil
}

// TierOf returns the highest-precedence tier that defines the key.
func (tp *TieredPersister) TierOf(key string) string {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	if tiers, ok := tp.tierMap[key]; ok && len(tiers) > 0 {
		return tiers[0]
	}
	return ""
}

// Tiers returns all tiers that define the key, highest precedence first.
func (tp *TieredPersister) Tiers(key string) []string {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	if tiers, ok := tp.tierMap[key]; ok {
		out := make([]string, len(tiers))
		copy(out, tiers)
		return out
	}
	return nil
}

// Watch monitors parent directories of all tier paths for external changes.
func (tp *TieredPersister) Watch(onChange func()) error {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if tp.watcher != nil {
		return nil
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}

	// track which dirs we've added to avoid duplicates
	watched := make(map[string]bool)
	for _, t := range tp.tiers {
		dir := filepath.Dir(t.Path)
		if watched[dir] {
			continue
		}
		// ensure parent dir exists so we can watch it
		if err := pkg.EnsureDir(dir); err != nil {
			w.Close()
			return fmt.Errorf("ensure dir %s: %w", dir, err)
		}
		if err := w.Add(dir); err != nil {
			w.Close()
			return fmt.Errorf("watch dir %s: %w", dir, err)
		}
		watched[dir] = true
	}

	tp.watcher = w

	// build set of filenames we care about
	watchedFiles := make(map[string]bool, len(tp.tiers))
	for _, t := range tp.tiers {
		watchedFiles[t.Path] = true
	}

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
				if !watchedFiles[ev.Name] {
					continue
				}

				// suppress self-write events
				tp.mu.Lock()
				selfTS := tp.selfWrite
				tp.mu.Unlock()
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
func (tp *TieredPersister) Unwatch() {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if tp.watcher != nil {
		tp.watcher.Close()
		tp.watcher = nil
	}
}

// --- internal helpers ---

// mergeAll reads all tier files and deep-merges them in precedence order.
// returns the merged map, and a tier provenance map (key → tiers, highest first).
func (tp *TieredPersister) mergeAll() (merged map[string]any, tierMap map[string][]string, err error) {
	merged = make(map[string]any)
	// tierKeys[tierName] → set of dotted keys defined in that tier
	tierKeys := make(map[string]map[string]bool, len(tp.tiers))

	for _, t := range tp.tiers {
		raw, readErr := readFileOrEmpty(t.Path)
		if readErr != nil {
			return nil, nil, fmt.Errorf("read tier %s (%s): %w", t.Name, t.Path, readErr)
		}
		if len(raw) == 0 {
			tierKeys[t.Name] = make(map[string]bool)
			continue
		}

		var tierObj map[string]any
		if parseErr := json.Unmarshal(raw, &tierObj); parseErr != nil {
			return nil, nil, fmt.Errorf("parse tier %s: %w", t.Name, parseErr)
		}

		keys := make(map[string]bool)
		collectDottedKeys(tierObj, nil, keys)
		tierKeys[t.Name] = keys

		deepMerge(merged, tierObj)
	}

	// build tierMap: for each key, list tiers that define it (highest precedence first)
	allKeys := make(map[string]bool)
	for _, keys := range tierKeys {
		for k := range keys {
			allKeys[k] = true
		}
	}

	tierMap = make(map[string][]string, len(allKeys))
	for key := range allKeys {
		var tiers []string
		// iterate in reverse (highest precedence first)
		for i := len(tp.tiers) - 1; i >= 0; i-- {
			if tierKeys[tp.tiers[i].Name][key] {
				tiers = append(tiers, tp.tiers[i].Name)
			}
		}
		tierMap[key] = tiers
	}

	return merged, tierMap, nil
}

// deepMerge merges src into dst. objects are recursively merged;
// arrays and scalars in src replace dst (atomic replace).
func deepMerge(dst, src map[string]any) {
	for k, sv := range src {
		dv, exists := dst[k]
		if !exists {
			dst[k] = sv
			continue
		}

		srcMap, srcIsMap := sv.(map[string]any)
		dstMap, dstIsMap := dv.(map[string]any)
		if srcIsMap && dstIsMap {
			deepMerge(dstMap, srcMap)
			continue
		}

		// atomic replace for arrays, scalars, or type mismatch
		dst[k] = sv
	}
}

// collectDottedKeys gathers all leaf dotted paths from a nested map.
func collectDottedKeys(m map[string]any, prefix []string, out map[string]bool) {
	for k, v := range m {
		full := append(append([]string{}, prefix...), k)
		if child, ok := v.(map[string]any); ok {
			collectDottedKeys(child, full, out)
		} else {
			out[strings.Join(full, ".")] = true
		}
	}
}

// routeKey determines which tier file a new or changed key should be written to.
func (tp *TieredPersister) routeKey(key string, tierMap map[string][]string) string {
	// if key already has provenance, route to its owning tier
	if tiers, ok := tierMap[key]; ok && len(tiers) > 0 {
		return tiers[0]
	}

	// for new keys: check if a sibling or parent key lives in a specific tier
	parts := strings.Split(key, ".")
	if len(parts) > 1 {
		// check parent path
		parentKey := strings.Join(parts[:len(parts)-1], ".")
		for k, tiers := range tierMap {
			if strings.HasPrefix(k, parentKey+".") && len(tiers) > 0 {
				return tiers[0]
			}
		}
	}

	// default: global
	return TierGlobal
}

// ownerTier returns the highest-precedence tier that owns the key.
func (tp *TieredPersister) ownerTier(key string, tierMap map[string][]string) string {
	if tiers, ok := tierMap[key]; ok && len(tiers) > 0 {
		return tiers[0]
	}
	return ""
}

// readFileOrEmpty reads a file, returning empty bytes if it doesn't exist.
func readFileOrEmpty(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

// marshalJSON produces pretty-printed JSON with 2-space indent and trailing newline.
func marshalJSON(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// listAllKeys returns all leaf dotted keys from a JSON blob.
func listAllKeys(data []byte) []string {
	p := &pkg.JSONParser{}
	return p.ListKeys(data)
}

// copyTierMap returns a shallow copy of a tier map.
func copyTierMap(m map[string][]string) map[string][]string {
	out := make(map[string][]string, len(m))
	for k, v := range m {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

// containsKey checks if a string slice contains a value.
func containsKey(keys []string, key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}
