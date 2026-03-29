package parser

import "strings"

// KeySplitter splits a dotted config key into section and field parts.
type KeySplitter func(key string) (section, field string)

// SplitKeyLast splits at the last dot: "a.b.c" → ("a.b", "c").
// use for TOML configs with dotted section headers (alacritty, helix, rio).
func SplitKeyLast(key string) (string, string) {
	idx := strings.LastIndexByte(key, '.')
	if idx < 0 {
		return "", key
	}
	return key[:idx], key[idx+1:]
}

// SplitKeyFirst splits at the first dot: "a.b" → ("a", "b").
// use for INI-style configs with flat section names (git, starship, pacman).
func SplitKeyFirst(key string) (string, string) {
	section, field, found := strings.Cut(key, ".")
	if !found {
		return "", key
	}
	return section, field
}

// SectionParser implements Parser for configs with [section] headers
// and key = value pairs (TOML, INI, git config).
type SectionParser struct {
	SplitKey     KeySplitter
	CommentChars string // characters that start a comment line, default "#"
}

func (p *SectionParser) isCommentStart(c byte) bool {
	chars := p.CommentChars
	if chars == "" {
		chars = "#"
	}
	return strings.IndexByte(chars, c) >= 0
}

func (p *SectionParser) FindValue(data []byte, key string) (string, bool) {
	section, field := p.SplitKey(key)
	if section == "" {
		val, _, found := FindTopLevelKey(data, field)
		return val, found
	}
	val, _, found := FindKeyInSection(data, section, field)
	return val, found
}

func (p *SectionParser) FindLine(data []byte, key string) (int, bool) {
	section, field := p.SplitKey(key)
	if section == "" {
		_, lineIdx, found := FindTopLevelKey(data, field)
		return lineIdx, found
	}
	_, lineIdx, found := FindKeyInSection(data, section, field)
	return lineIdx, found
}

func (p *SectionParser) SetValue(data []byte, key, value string) ([]byte, error) {
	section, field := p.SplitKey(key)

	if section == "" {
		_, lineIdx, found := FindTopLevelKey(data, field)
		if found {
			return ReplaceValueOnLine(data, lineIdx, value), nil
		}
		return InsertTopLevelKey(data, field, value), nil
	}

	_, lineIdx, found := FindKeyInSection(data, section, field)
	if found {
		return ReplaceValueOnLine(data, lineIdx, value), nil
	}
	return InsertKeyInSection(data, section, field, value), nil
}

func (p *SectionParser) DeleteKey(data []byte, key string) ([]byte, error) {
	section, field := p.SplitKey(key)

	if section == "" {
		_, lineIdx, found := FindTopLevelKey(data, field)
		if !found {
			return data, nil
		}
		return DeleteKeyOnLine(data, lineIdx), nil
	}

	_, lineIdx, found := FindKeyInSection(data, section, field)
	if !found {
		return data, nil
	}
	return DeleteKeyOnLine(data, lineIdx), nil
}

// FindAll returns all key-value pairs in a single pass, using dotted keys.
func (p *SectionParser) FindAll(data []byte) map[string]string {
	lines := strings.Split(string(data), "\n")
	m := make(map[string]string)
	currentSection := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || p.isCommentStart(trimmed[0]) {
			continue
		}
		if trimmed[0] == '[' {
			currentSection = ParseSectionHeader(trimmed)
			continue
		}
		k, v, ok := ParseKVLine(trimmed)
		if !ok {
			continue
		}
		if currentSection != "" {
			m[currentSection+"."+k] = v
		} else {
			m[k] = v
		}
	}
	return m
}

func (p *SectionParser) ListKeys(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	var keys []string
	currentSection := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || p.isCommentStart(trimmed[0]) {
			continue
		}
		if trimmed[0] == '[' {
			currentSection = ParseSectionHeader(trimmed)
			continue
		}
		k, _, ok := ParseKVLine(trimmed)
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
