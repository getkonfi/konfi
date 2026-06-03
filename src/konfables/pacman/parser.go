package pacman

import (
	"strings"

	cfgparse "github.com/eminert/konfi/pkg/parser"
)

// parser handles pacman.conf INI format: [section] headers, key = value pairs,
// and bare directives (presence-based boolean flags like Color, ILoveCandy).
type parser struct {
	base cfgparse.SectionParser
}

func newParser() *parser {
	return &parser{base: cfgparse.SectionParser{SplitKey: cfgparse.SplitKeyFirst}}
}

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
	val, found := p.base.FindValue(data, key)
	if found {
		return val, true
	}
	section, field := cfgparse.SplitKeyFirst(key)
	if section != "" {
		if _, found := findBareDirective(data, section, field); found {
			return "true", true
		}
	}
	return "", false
}

// FindAll returns all key-value pairs in a single pass, including bare directives.
func (p *parser) FindAll(data []byte) map[string]string {
	lines := strings.Split(string(data), "\n")
	m := make(map[string]string)
	currentSection := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed[0] == '#' {
			continue
		}
		if trimmed[0] == '[' {
			currentSection = cfgparse.ParseSectionHeader(trimmed)
			continue
		}
		k, v, ok := cfgparse.ParseKVLine(trimmed)
		if !ok {
			// bare directive
			if isValidDirective(trimmed) {
				k = trimmed
				v = "true"
			} else {
				continue
			}
		}
		if currentSection != "" {
			m[currentSection+"."+k] = v
		} else {
			m[k] = v
		}
	}
	return m
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	lineIdx, found := p.base.FindLine(data, key)
	if found {
		return lineIdx, true
	}
	section, field := cfgparse.SplitKeyFirst(key)
	if section != "" {
		return findBareDirective(data, section, field)
	}
	return -1, false
}

func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	_, field := cfgparse.SplitKeyFirst(key)
	if bareDirectives[field] {
		section, _ := cfgparse.SplitKeyFirst(key)
		return setBareDirective(data, section, field, value)
	}
	return p.base.SetValue(data, key, value)
}

func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	section, field := cfgparse.SplitKeyFirst(key)
	if section == "" {
		return p.base.DeleteKey(data, key)
	}
	_, lineIdx, found := cfgparse.FindKeyInSection(data, section, field)
	if found {
		return cfgparse.DeleteKeyOnLine(data, lineIdx), nil
	}
	lineIdx, found = findBareDirective(data, section, field)
	if found {
		return cfgparse.DeleteKeyOnLine(data, lineIdx), nil
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
			currentSection = cfgparse.ParseSectionHeader(trimmed)
			continue
		}
		k, _, ok := cfgparse.ParseKVLine(trimmed)
		if !ok {
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

func findBareDirective(data []byte, section, key string) (int, bool) {
	lines := strings.Split(string(data), "\n")
	inSection := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && trimmed[0] == '[' {
			sectionName := cfgparse.ParseSectionHeader(trimmed)
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

func setBareDirective(data []byte, section, field, value string) ([]byte, error) {
	lineIdx, found := findBareDirective(data, section, field)

	if value == "false" {
		if found {
			return cfgparse.DeleteKeyOnLine(data, lineIdx), nil
		}
		return data, nil
	}

	if found {
		return data, nil
	}
	_, kvIdx, kvFound := cfgparse.FindKeyInSection(data, section, field)
	if kvFound {
		data = cfgparse.DeleteKeyOnLine(data, kvIdx)
	}
	return insertBareInSection(data, section, field), nil
}

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
			sectionName := cfgparse.ParseSectionHeader(trimmed)
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
