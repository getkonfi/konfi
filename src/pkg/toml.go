package pkg

import (
	"strings"
)

// line-level TOML helpers for surgical config editing.
// operates on raw bytes, splits on \n, rejoins.
// preserves comments, whitespace, key ordering.

// FindTopLevelKey finds a key before any section header.
func FindTopLevelKey(data []byte, key string) (value string, lineIdx int, found bool) {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed[0] == '#' {
			continue
		}
		// stop at first section
		if trimmed[0] == '[' {
			break
		}
		k, v, ok := ParseKVLine(trimmed)
		if ok && k == key {
			return v, i, true
		}
	}
	return "", -1, false
}

// FindKeyInSection finds a key within a [section] block.
func FindKeyInSection(data []byte, section, key string) (value string, lineIdx int, found bool) {
	lines := strings.Split(string(data), "\n")
	inSection := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// section header
		if trimmed != "" && trimmed[0] == '[' {
			sectionName := ParseSectionHeader(trimmed)
			inSection = sectionName == section
			continue
		}

		if !inSection {
			continue
		}

		if trimmed == "" || trimmed[0] == '#' {
			continue
		}

		k, v, ok := ParseKVLine(trimmed)
		if ok && k == key {
			return v, i, true
		}
	}
	return "", -1, false
}

// ReplaceValueOnLine replaces the value portion of a key = value line.
func ReplaceValueOnLine(data []byte, lineIdx int, newValue string) []byte {
	lines := strings.Split(string(data), "\n")
	if lineIdx < 0 || lineIdx >= len(lines) {
		return data
	}

	line := lines[lineIdx]
	eqIdx := strings.IndexByte(line, '=')
	if eqIdx < 0 {
		return data
	}

	// preserve key and spacing before =
	prefix := line[:eqIdx+1]

	// preserve one space after = if the original had it
	rest := line[eqIdx+1:]
	if rest != "" && rest[0] == ' ' {
		prefix += " "
	}

	// strip inline comment from original for replacement
	lines[lineIdx] = prefix + newValue
	return []byte(strings.Join(lines, "\n"))
}

// InsertKeyInSection adds a key-value pair at the end of a section.
// if the section doesn't exist, it creates it at the end of the file.
func InsertKeyInSection(data []byte, section, key, value string) []byte {
	lines := strings.Split(string(data), "\n")
	sectionEnd := -1
	inSection := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && trimmed[0] == '[' {
			if inSection {
				// found next section, insert before it
				return insertLine(lines, i, key+" = "+value)
			}
			sectionName := ParseSectionHeader(trimmed)
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
		// section found, insert after last non-empty line
		return insertLine(lines, sectionEnd+1, key+" = "+value)
	}

	// section not found, append new section
	result := string(data)
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	result += "\n[" + section + "]\n" + key + " = " + value + "\n"
	return []byte(result)
}

// InsertTopLevelKey adds a key-value pair before any section.
func InsertTopLevelKey(data []byte, key, value string) []byte {
	lines := strings.Split(string(data), "\n")

	// find first section header
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && trimmed[0] == '[' {
			return insertLine(lines, i, key+" = "+value)
		}
	}

	// no sections, append
	return insertLine(lines, len(lines), key+" = "+value)
}

// DeleteKeyOnLine removes a line entirely.
func DeleteKeyOnLine(data []byte, lineIdx int) []byte {
	lines := strings.Split(string(data), "\n")
	if lineIdx < 0 || lineIdx >= len(lines) {
		return data
	}
	lines = append(lines[:lineIdx], lines[lineIdx+1:]...)
	return []byte(strings.Join(lines, "\n"))
}

// --- helpers ---

// ParseSectionHeader extracts the section name from a [section] line.
func ParseSectionHeader(line string) string {
	line = strings.TrimSpace(line)
	if len(line) < 2 || line[0] != '[' {
		return ""
	}
	end := strings.IndexByte(line, ']')
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(line[1:end])
}

// ParseKVLine extracts key and value from a "key = value" line.
func ParseKVLine(line string) (key, value string, ok bool) {
	eqIdx := strings.IndexByte(line, '=')
	if eqIdx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:eqIdx])
	value = strings.TrimSpace(line[eqIdx+1:])

	// strip quotes from value
	if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"') {
		value = value[1 : len(value)-1]
	}

	return key, value, true
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
