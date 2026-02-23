package starship

import (
	"strings"

	"github.com/emin/konfigurator/pkg"
)

type parser struct{}

// FindValue looks up a key in starship TOML.
// keys use "section.field" notation; bare keys are top-level.
func (p *parser) FindValue(data []byte, key string) (string, bool) {
	section, field, nested := splitKey(key)
	if !nested {
		val, _, found := pkg.FindTopLevelKey(data, key)
		return val, found
	}
	val, _, found := pkg.FindKeyInSection(data, section, field)
	return val, found
}

// FindLine returns the 0-based line index where key is defined.
func (p *parser) FindLine(data []byte, key string) (int, bool) {
	section, field, nested := splitKey(key)
	if !nested {
		_, lineIdx, found := pkg.FindTopLevelKey(data, key)
		return lineIdx, found
	}
	_, lineIdx, found := pkg.FindKeyInSection(data, section, field)
	return lineIdx, found
}

// SetValue sets a key to value, replacing if it exists or inserting if not.
func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	section, field, nested := splitKey(key)

	if !nested {
		_, lineIdx, found := pkg.FindTopLevelKey(data, key)
		if found {
			return pkg.ReplaceValueOnLine(data, lineIdx, value), nil
		}
		return pkg.InsertTopLevelKey(data, key, value), nil
	}

	_, lineIdx, found := pkg.FindKeyInSection(data, section, field)
	if found {
		return pkg.ReplaceValueOnLine(data, lineIdx, value), nil
	}
	return pkg.InsertKeyInSection(data, section, field, value), nil
}

// DeleteKey removes a key from the config. returns data unchanged if not found.
func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	section, field, nested := splitKey(key)

	if !nested {
		_, lineIdx, found := pkg.FindTopLevelKey(data, key)
		if !found {
			return data, nil
		}
		return pkg.DeleteKeyOnLine(data, lineIdx), nil
	}

	_, lineIdx, found := pkg.FindKeyInSection(data, section, field)
	if !found {
		return data, nil
	}
	return pkg.DeleteKeyOnLine(data, lineIdx), nil
}

// ListKeys returns all config keys defined in the data as dotted paths.
func (p *parser) ListKeys(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	var keys []string
	currentSection := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed[0] == '#' {
			continue
		}
		if trimmed[0] == '[' {
			currentSection = pkg.ParseSectionHeader(trimmed)
			continue
		}
		k, _, ok := pkg.ParseKVLine(trimmed)
		if !ok {
			continue
		}
		if currentSection != "" {
			keys = append(keys, currentSection+"."+k)
		} else {
			keys = append(keys, k)
		}
	}
	return keys
}

// splitKey splits "section.field" into parts. returns nested=false for bare keys.
func splitKey(key string) (section, field string, nested bool) {
	idx := strings.IndexByte(key, '.')
	if idx < 0 {
		return "", key, false
	}
	return key[:idx], key[idx+1:], true
}
