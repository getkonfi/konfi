package ghostty

import (
	"bytes"
	"strings"
)

type parser struct{}

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	v, _, found := find(data, key)
	return v, found
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	_, i, found := find(data, key)
	return i, found
}

// SetValue replaces an existing key's value or appends a new line.
func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	lines := bytes.Split(data, []byte("\n"))
	found := false
	for i, line := range lines {
		s := strings.TrimSpace(string(line))
		if s == "" || s[0] == '#' {
			continue
		}
		k, _, ok := splitKV(s)
		if ok && k == key {
			lines[i] = []byte(key + " = " + value)
			found = true
			break
		}
	}
	if !found {
		// ensure we append on a new line
		if len(data) > 0 && data[len(data)-1] != '\n' {
			lines = append(lines, []byte(key+" = "+value))
		} else {
			// data ended with newline, so Split produced a trailing empty element
			lines = append(lines[:len(lines)-1], []byte(key+" = "+value), []byte(""))
		}
	}
	return bytes.Join(lines, []byte("\n")), nil
}

// find scans lines for key, skipping comments and blanks.
func find(data []byte, key string) (value string, lineIdx int, found bool) {
	lines := bytes.Split(data, []byte("\n"))
	for i, line := range lines {
		s := strings.TrimSpace(string(line))
		if s == "" || s[0] == '#' {
			continue
		}
		k, v, ok := splitKV(s)
		if ok && k == key {
			return v, i, true
		}
	}
	return "", -1, false
}

// DeleteKey removes the line containing the key.
func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	lines := bytes.Split(data, []byte("\n"))
	out := make([][]byte, 0, len(lines))
	for _, line := range lines {
		s := strings.TrimSpace(string(line))
		if s != "" && s[0] != '#' {
			k, _, ok := splitKV(s)
			if ok && k == key {
				continue
			}
		}
		out = append(out, line)
	}
	return bytes.Join(out, []byte("\n")), nil
}

// FindValues collects all values for a repeated key (e.g., keybind, palette).
func (p *parser) FindValues(data []byte, key string) ([]string, bool) {
	lines := bytes.Split(data, []byte("\n"))
	var vals []string
	for _, line := range lines {
		s := strings.TrimSpace(string(line))
		if s == "" || s[0] == '#' {
			continue
		}
		k, v, ok := splitKV(s)
		if ok && k == key {
			vals = append(vals, v)
		}
	}
	if len(vals) == 0 {
		return nil, false
	}
	return vals, true
}

// SetValues replaces all instances of a repeated key with the given values.
func (p *parser) SetValues(data []byte, key string, values []string) ([]byte, error) {
	lines := bytes.Split(data, []byte("\n"))
	// remove all existing lines for this key
	out := make([][]byte, 0, len(lines))
	for _, line := range lines {
		s := strings.TrimSpace(string(line))
		if s != "" && s[0] != '#' {
			k, _, ok := splitKV(s)
			if ok && k == key {
				continue
			}
		}
		out = append(out, line)
	}
	// append new values
	for _, v := range values {
		newLine := []byte(key + " = " + v)
		// insert before trailing empty line if present
		if len(out) > 0 && len(bytes.TrimSpace(out[len(out)-1])) == 0 {
			out = append(out[:len(out)-1], newLine, out[len(out)-1])
		} else {
			out = append(out, newLine)
		}
	}
	return bytes.Join(out, []byte("\n")), nil
}

// ListKeys returns all config keys defined in the data.
func (p *parser) ListKeys(data []byte) []string {
	lines := bytes.Split(data, []byte("\n"))
	var keys []string
	for _, line := range lines {
		s := strings.TrimSpace(string(line))
		if s == "" || s[0] == '#' {
			continue
		}
		k, _, ok := splitKV(s)
		if ok {
			keys = append(keys, k)
		}
	}
	return keys
}

// splitKV parses "key = value" returning trimmed key, value, ok.
func splitKV(s string) (key, value string, ok bool) {
	idx := strings.Index(s, "=")
	if idx < 0 {
		return "", "", false
	}
	return strings.TrimSpace(s[:idx]), strings.TrimSpace(s[idx+1:]), true
}
