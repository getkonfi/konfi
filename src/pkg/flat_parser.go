package pkg

import (
	"strings"
)

// LineSplitter extracts key and value from a config line.
type LineSplitter func(s string) (key, value string, ok bool)

// LineFormatter formats a key-value pair for writing.
type LineFormatter func(key, value string) string

// SplitEquals splits on "=" and trims whitespace: "key = value" → ("key", "value").
func SplitEquals(s string) (string, string, bool) {
	k, v, found := strings.Cut(s, "=")
	if !found {
		return "", "", false
	}
	return strings.TrimSpace(k), strings.TrimSpace(v), true
}

// SplitEqualsOrSpace splits on "=" first, falling back to space-separated "key value".
func SplitEqualsOrSpace(s string) (string, string, bool) {
	if k, v, found := strings.Cut(s, "="); found {
		k = strings.TrimSpace(k)
		if k != "" && !strings.ContainsRune(k, ' ') {
			return k, strings.TrimSpace(v), true
		}
	}
	parts := strings.SplitN(s, " ", 2)
	if len(parts) < 1 || parts[0] == "" {
		return "", "", false
	}
	if len(parts) == 1 {
		return parts[0], "", true
	}
	return parts[0], strings.TrimSpace(parts[1]), true
}

// SplitColon splits on ":" and trims: "key: value" → ("key", "value").
func SplitColon(s string) (string, string, bool) {
	k, v, found := strings.Cut(s, ":")
	if !found {
		return "", "", false
	}
	return strings.TrimSpace(k), strings.TrimSpace(v), true
}

// SplitSpacedEquals splits on " = " (space-equals-space): "key = value" → ("key", "value").
func SplitSpacedEquals(s string) (string, string, bool) {
	k, v, found := strings.Cut(s, " = ")
	if !found {
		return "", "", false
	}
	return k, v, true
}

// FormatEquals formats as "key = value".
func FormatEquals(key, value string) string { return key + " = " + value }

// FormatSpace formats as "key value".
func FormatSpace(key, value string) string { return key + " " + value }

// FormatColon formats as "key: value".
func FormatColon(key, value string) string { return key + ": " + value }

// FlatParser implements Parser for flat key-value configs (no sections).
// also implements MultiValueParser when used with repeated-key formats.
type FlatParser struct {
	Split  LineSplitter
	Format LineFormatter
}

func (p *FlatParser) FindValue(data []byte, key string) (string, bool) {
	v, _, found := p.find(data, key)
	return v, found
}

func (p *FlatParser) FindLine(data []byte, key string) (int, bool) {
	_, i, found := p.find(data, key)
	return i, found
}

func (p *FlatParser) find(data []byte, key string) (value string, lineIdx int, found bool) {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" || s[0] == '#' {
			continue
		}
		k, v, ok := p.Split(s)
		if ok && k == key {
			return v, i, true
		}
	}
	return "", -1, false
}

func (p *FlatParser) SetValue(data []byte, key, value string) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" || s[0] == '#' {
			continue
		}
		k, oldVal, ok := p.Split(s)
		if ok && k == key {
			// preserve original line format by replacing only the value portion
			lead := len(line) - len(strings.TrimLeft(line, " \t"))
			if oldVal != "" {
				// value sits at end of trimmed content
				valStart := lead + len(s) - len(oldVal)
				lines[i] = line[:valStart] + value
			} else {
				// empty old value — append after existing content
				lines[i] = strings.TrimRight(line, " \t") + value
			}
			return []byte(strings.Join(lines, "\n")), nil
		}
	}
	// not found — append
	if len(data) > 0 && data[len(data)-1] != '\n' {
		lines = append(lines, p.Format(key, value))
	} else {
		lines = append(lines[:len(lines)-1], p.Format(key, value), "")
	}
	return []byte(strings.Join(lines, "\n")), nil
}

func (p *FlatParser) DeleteKey(data []byte, key string) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s != "" && s[0] != '#' {
			k, _, ok := p.Split(s)
			if ok && k == key {
				continue
			}
		}
		out = append(out, line)
	}
	return []byte(strings.Join(out, "\n")), nil
}

// FindAll returns all key-value pairs in a single pass.
func (p *FlatParser) FindAll(data []byte) map[string]string {
	lines := strings.Split(string(data), "\n")
	m := make(map[string]string, len(lines)/2)
	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" || s[0] == '#' {
			continue
		}
		k, v, ok := p.Split(s)
		if ok {
			m[k] = v
		}
	}
	return m
}

// FindAllMulti returns all key-value pairs in a single pass,
// collecting repeated keys into slices.
func (p *FlatParser) FindAllMulti(data []byte) (singles map[string]string, multi map[string][]string) {
	lines := strings.Split(string(data), "\n")
	singles = make(map[string]string, len(lines)/2)
	count := make(map[string]int, len(lines)/2)
	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" || s[0] == '#' {
			continue
		}
		k, v, ok := p.Split(s)
		if !ok {
			continue
		}
		count[k]++
		if count[k] == 1 {
			singles[k] = v
		} else if count[k] == 2 {
			// promote to multi
			if multi == nil {
				multi = make(map[string][]string)
			}
			multi[k] = []string{singles[k], v}
			delete(singles, k)
		} else {
			multi[k] = append(multi[k], v)
		}
	}
	return singles, multi
}

func (p *FlatParser) ListKeys(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	var keys []string
	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" || s[0] == '#' {
			continue
		}
		k, _, ok := p.Split(s)
		if ok {
			keys = append(keys, k)
		}
	}
	return keys
}

// FindValues collects all values for a repeated key (e.g., ghostty keybind, palette).
func (p *FlatParser) FindValues(data []byte, key string) ([]string, bool) {
	lines := strings.Split(string(data), "\n")
	var vals []string
	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" || s[0] == '#' {
			continue
		}
		k, v, ok := p.Split(s)
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
func (p *FlatParser) SetValues(data []byte, key string, values []string) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s != "" && s[0] != '#' {
			k, _, ok := p.Split(s)
			if ok && k == key {
				continue
			}
		}
		out = append(out, line)
	}
	for _, v := range values {
		newLine := p.Format(key, v)
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
			out = append(out[:len(out)-1], newLine, out[len(out)-1])
		} else {
			out = append(out, newLine)
		}
	}
	return []byte(strings.Join(out, "\n")), nil
}
