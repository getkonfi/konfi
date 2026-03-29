package pkg

import (
	"sort"
	"strings"
	"unicode"
)

// CrossRefMatch describes a field in another app that is semantically similar.
type CrossRefMatch struct {
	App   string
	Key   string
	Label string
}

// crossRefEntry is an internal entry in the cross-reference index.
type crossRefEntry struct {
	app    string
	key    string
	label  string
	tokens map[string]bool
}

// CrossRefIndex enables finding equivalent fields across installed apps.
type CrossRefIndex struct {
	entries []crossRefEntry
}

// NewCrossRefIndex builds a cross-reference index from parsed schemas.
// only installed apps are indexed.
func NewCrossRefIndex(schemas map[string]*Schema, installed []string) *CrossRefIndex {
	installedSet := make(map[string]bool, len(installed))
	for _, name := range installed {
		installedSet[name] = true
	}

	var entries []crossRefEntry
	for name, s := range schemas {
		if !installedSet[name] {
			continue
		}
		for si := range s.Sections {
			for fi := range s.Sections[si].Fields {
				f := &s.Sections[si].Fields[fi]
				tokens := crossRefTokenize(f.Key + " " + f.Label)
				if len(tokens) == 0 {
					continue
				}
				entries = append(entries, crossRefEntry{
					app:    name,
					key:    f.Key,
					label:  f.Label,
					tokens: tokens,
				})
			}
		}
	}
	return &CrossRefIndex{entries: entries}
}

// FindEquivalents returns fields in other apps that share tokens with the given field.
func (idx *CrossRefIndex) FindEquivalents(app, key, label string, limit int) []CrossRefMatch {
	if idx == nil || len(idx.entries) == 0 {
		return nil
	}
	query := crossRefTokenize(key + " " + label)
	if len(query) == 0 {
		return nil
	}

	type scored struct {
		entry crossRefEntry
		score int
	}
	var candidates []scored

	for i := range idx.entries {
		e := &idx.entries[i]
		if e.app == app {
			continue
		}
		overlap := 0
		for t := range query {
			if e.tokens[t] {
				overlap++
			}
		}
		if overlap < 1 {
			continue
		}
		candidates = append(candidates, scored{entry: *e, score: overlap})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	matches := make([]CrossRefMatch, len(candidates))
	for i, c := range candidates {
		matches[i] = CrossRefMatch{
			App:   c.entry.app,
			Key:   c.entry.key,
			Label: c.entry.label,
		}
	}
	return matches
}

// crossRefTokenize splits a string on delimiters and camelCase boundaries, normalizes to lowercase.
func crossRefTokenize(s string) map[string]bool {
	tokens := make(map[string]bool)

	// split on common delimiters
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == '.' || r == ' ' || r == '/' || r == ':'
	})

	for _, part := range parts {
		// split camelCase
		for _, sub := range splitCamel(part) {
			t := strings.ToLower(sub)
			if len(t) > 1 { // skip single-char tokens
				tokens[t] = true
			}
		}
	}
	return tokens
}

// splitCamel splits a camelCase or PascalCase string into words.
func splitCamel(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	runes := []rune(s)
	start := 0
	for i := 1; i < len(runes); i++ {
		if unicode.IsUpper(runes[i]) && !unicode.IsUpper(runes[i-1]) {
			result = append(result, string(runes[start:i]))
			start = i
		}
	}
	result = append(result, string(runes[start:]))
	return result
}
