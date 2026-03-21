package pacman

import (
	"strings"

	"github.com/emin/konfigurator/pkg"
)

// parser handles pacman.conf INI format: [section] headers, key = value pairs,
// and bare directives (presence-based boolean flags like Color, ILoveCandy).
// reuses the TOML line-level helpers for key=value lines.
type parser struct{}

// bareDirectives lists pacman options that are presence-based boolean flags.
var bareDirectives = map[string]bool{
	"UseSyslog":              true,
	"Color":                  true,
	"NoProgressBar":          true,
	"CheckSpace":             true,
	"VerbosePkgLists":        true,
	"DisableDownloadTimeout": true,
	"ILoveCandy":             true,
}

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	section, field := splitKey(key)
	if section == "" {
		val, _, found := pkg.FindTopLevelKey(data, field)
		return val, found
	}
	// try key=value first
	val, _, found := pkg.FindKeyInSection(data, section, field)
	if found {
		return val, true
	}
	// try bare directive
	if _, found := findBareDirective(data, section, field); found {
		return "true", true
	}
	return "", false
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	section, field := splitKey(key)
	if section == "" {
		_, lineIdx, found := pkg.FindTopLevelKey(data, field)
		return lineIdx, found
	}
	_, lineIdx, found := pkg.FindKeyInSection(data, section, field)
	if found {
		return lineIdx, true
	}
	lineIdx, found = findBareDirective(data, section, field)
	return lineIdx, found
}

func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	section, field := splitKey(key)

	if bareDirectives[field] {
		return setBareDirective(data, section, field, value)
	}

	// normal key=value handling
	if section == "" {
		_, lineIdx, found := pkg.FindTopLevelKey(data, field)
		if found {
			return pkg.ReplaceValueOnLine(data, lineIdx, value), nil
		}
		return pkg.InsertTopLevelKey(data, field, value), nil
	}

	_, lineIdx, found := pkg.FindKeyInSection(data, section, field)
	if found {
		return pkg.ReplaceValueOnLine(data, lineIdx, value), nil
	}
	return pkg.InsertKeyInSection(data, section, field, value), nil
}

func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	section, field := splitKey(key)

	if section == "" {
		_, lineIdx, found := pkg.FindTopLevelKey(data, field)
		if !found {
			return data, nil
		}
		return pkg.DeleteKeyOnLine(data, lineIdx), nil
	}

	// try key=value
	_, lineIdx, found := pkg.FindKeyInSection(data, section, field)
	if found {
		return pkg.DeleteKeyOnLine(data, lineIdx), nil
	}
	// try bare directive
	lineIdx, found = findBareDirective(data, section, field)
	if found {
		return pkg.DeleteKeyOnLine(data, lineIdx), nil
	}
	return data, nil
}

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
			// bare directive: a single word with no = sign
			if isValidDirective(trimmed) {
				k = trimmed
			} else {
				continue
			}
		}
		if currentSection != "" {
			keys = append(keys, currentSection+"."+k)
		} else {
			keys = append(keys, k)
		}
	}
	return keys
}

// findBareDirective searches for a bare directive (no = sign) within a section.
func findBareDirective(data []byte, section, key string) (int, bool) {
	lines := strings.Split(string(data), "\n")
	inSection := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && trimmed[0] == '[' {
			sectionName := pkg.ParseSectionHeader(trimmed)
			inSection = sectionName == section
			continue
		}
		if !inSection {
			continue
		}
		if trimmed == "" || trimmed[0] == '#' {
			continue
		}
		if trimmed == key {
			return i, true
		}
	}
	return -1, false
}

// setBareDirective handles set/unset for presence-based boolean flags.
func setBareDirective(data []byte, section, field, value string) ([]byte, error) {
	lineIdx, found := findBareDirective(data, section, field)

	if value == "false" {
		if found {
			return pkg.DeleteKeyOnLine(data, lineIdx), nil
		}
		return data, nil
	}

	// value is "true" or anything else — ensure directive is present
	if found {
		return data, nil
	}
	// also check if it exists as key=value and remove it first
	_, kvIdx, kvFound := pkg.FindKeyInSection(data, section, field)
	if kvFound {
		data = pkg.DeleteKeyOnLine(data, kvIdx)
	}
	return insertBareInSection(data, section, field), nil
}

// insertBareInSection adds a bare directive at the end of a section.
func insertBareInSection(data []byte, section, field string) []byte {
	lines := strings.Split(string(data), "\n")
	sectionEnd := -1
	inSection := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && trimmed[0] == '[' {
			if inSection {
				return insertLine(lines, i, field)
			}
			sectionName := pkg.ParseSectionHeader(trimmed)
			inSection = sectionName == section
			if inSection {
				sectionEnd = i
			}
			continue
		}
		if inSection && trimmed != "" {
			sectionEnd = i
		}
	}

	if inSection && sectionEnd >= 0 {
		return insertLine(lines, sectionEnd+1, field)
	}

	// section not found, append
	result := string(data)
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	result += "\n[" + section + "]\n" + field + "\n"
	return []byte(result)
}

func insertLine(lines []string, at int, content string) []byte {
	if at > len(lines) {
		at = len(lines)
	}
	result := make([]string, 0, len(lines)+1)
	result = append(result, lines[:at]...)
	result = append(result, content)
	result = append(result, lines[at:]...)
	return []byte(strings.Join(result, "\n"))
}

// isValidDirective checks if a line looks like a bare directive (single word, no special chars).
func isValidDirective(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c == ' ' || c == '\t' || c == '=' {
			return false
		}
	}
	return true
}

func splitKey(key string) (section, field string) {
	idx := strings.IndexByte(key, '.')
	if idx < 0 {
		return "", key
	}
	return key[:idx], key[idx+1:]
}
