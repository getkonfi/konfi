package dconf

import (
	"bytes"
	"fmt"
	"strings"
)

// parser operates on synthetic "path/key = value" flat bytes
// produced by DconfPersister.Load.
type parser struct{}

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	for _, line := range bytes.Split(data, []byte("\n")) {
		s := strings.TrimSpace(string(line))
		if s == "" || s[0] == '#' {
			continue
		}
		k, v, ok := cutKV(s)
		if ok && k == key {
			return v, true
		}
	}
	return "", false
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	for i, line := range bytes.Split(data, []byte("\n")) {
		s := strings.TrimSpace(string(line))
		if s == "" || s[0] == '#' {
			continue
		}
		k, _, ok := cutKV(s)
		if ok && k == key {
			return i, true
		}
	}
	return -1, false
}

func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	lines := bytes.Split(data, []byte("\n"))
	for i, line := range lines {
		s := strings.TrimSpace(string(line))
		if s == "" || s[0] == '#' {
			continue
		}
		k, _, ok := cutKV(s)
		if ok && k == key {
			lines[i] = []byte(fmt.Sprintf("%s = %s", key, value))
			return bytes.Join(lines, []byte("\n")), nil
		}
	}
	// not found — append
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	return append(data, []byte(fmt.Sprintf("%s = %s\n", key, value))...), nil
}

func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	lines := bytes.Split(data, []byte("\n"))
	out := make([][]byte, 0, len(lines))
	found := false
	for _, line := range lines {
		s := strings.TrimSpace(string(line))
		if s != "" && s[0] != '#' {
			k, _, ok := cutKV(s)
			if ok && k == key {
				found = true
				continue
			}
		}
		out = append(out, line)
	}
	if !found {
		return data, fmt.Errorf("key not found: %s", key)
	}
	return bytes.Join(out, []byte("\n")), nil
}

func (p *parser) ListKeys(data []byte) []string {
	var keys []string
	for _, line := range bytes.Split(data, []byte("\n")) {
		s := strings.TrimSpace(string(line))
		if s == "" || s[0] == '#' {
			continue
		}
		k, _, ok := cutKV(s)
		if ok {
			keys = append(keys, k)
		}
	}
	return keys
}

// cutKV splits "key = value" into (key, value, true).
func cutKV(s string) (string, string, bool) {
	idx := strings.Index(s, " = ")
	if idx < 0 {
		return "", "", false
	}
	return s[:idx], s[idx+3:], true
}
