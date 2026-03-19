package tmux

import (
	"strings"
)

// parser handles tmux config format: `set -g key value` / `set-option -g key value`.
type parser struct{}

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		k, v, ok := parseTmuxSet(line)
		if ok && k == key {
			return v, true
		}
	}
	return "", false
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		k, _, ok := parseTmuxSet(line)
		if ok && k == key {
			return i, true
		}
	}
	return -1, false
}

func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		k, _, ok := parseTmuxSet(line)
		if ok && k == key {
			lines[i] = replaceTmuxValue(line, value)
			return []byte(strings.Join(lines, "\n")), nil
		}
	}
	// not found — append
	newLine := "set -g " + key + " " + value
	result := string(data)
	if result != "" && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	result += newLine + "\n"
	return []byte(result), nil
}

func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		k, _, ok := parseTmuxSet(line)
		if ok && k == key {
			lines = append(lines[:i], lines[i+1:]...)
			return []byte(strings.Join(lines, "\n")), nil
		}
	}
	return data, nil
}

func (p *parser) ListKeys(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	var keys []string
	for _, line := range lines {
		k, _, ok := parseTmuxSet(line)
		if ok {
			keys = append(keys, k)
		}
	}
	return keys
}

// parseTmuxSet extracts key and value from a tmux set line.
// handles: set -g key value, set-option -g key value, setw -g key value
func parseTmuxSet(line string) (key, value string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed[0] == '#' {
		return "", "", false
	}

	fields := strings.Fields(trimmed)
	if len(fields) < 3 {
		return "", "", false
	}

	cmd := fields[0]
	if cmd != "set" && cmd != "set-option" && cmd != "set-window-option" && cmd != "setw" {
		return "", "", false
	}

	// skip flags (tokens starting with -)
	i := 1
	for i < len(fields) && strings.HasPrefix(fields[i], "-") {
		i++
	}

	if i >= len(fields) {
		return "", "", false
	}

	key = fields[i]
	i++

	if i < len(fields) {
		value = strings.Join(fields[i:], " ")
		value = stripQuotes(value)
	}

	return key, value, true
}

// replaceTmuxValue replaces the value portion of a tmux set line,
// preserving the command prefix and flags.
func replaceTmuxValue(line, newValue string) string {
	trimmed := strings.TrimSpace(line)
	fields := strings.Fields(trimmed)
	if len(fields) < 3 {
		return line
	}

	// find the key position (skip command + flags)
	i := 1
	for i < len(fields) && strings.HasPrefix(fields[i], "-") {
		i++
	}
	if i >= len(fields) {
		return line
	}

	// reconstruct: command + flags + key + newValue
	parts := fields[:i+1]
	return strings.Join(parts, " ") + " " + newValue
}

func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
