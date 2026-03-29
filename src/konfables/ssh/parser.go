package ssh

import (
	"strings"
)

// parser handles SSH config format: Keyword Value (space-separated, case-insensitive).
// scopes to global settings (before any Host block) and Host * blocks.
type parser struct{}

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	lines := strings.Split(string(data), "\n")
	scope := scopeGlobal
	for _, line := range lines {
		scope = updateScope(line, scope)
		if scope == scopeOther {
			continue
		}
		k, v, ok := parseSSHLine(line)
		if ok && strings.EqualFold(k, key) {
			return v, true
		}
	}
	return "", false
}

// FindAll returns all key-value pairs from global and Host * scopes in a single pass.
func (p *parser) FindAll(data []byte) map[string]string {
	lines := strings.Split(string(data), "\n")
	m := make(map[string]string)
	scope := scopeGlobal
	for _, line := range lines {
		scope = updateScope(line, scope)
		if scope == scopeOther {
			continue
		}
		k, v, ok := parseSSHLine(line)
		if ok {
			m[k] = v
		}
	}
	return m
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	lines := strings.Split(string(data), "\n")
	scope := scopeGlobal
	for i, line := range lines {
		scope = updateScope(line, scope)
		if scope == scopeOther {
			continue
		}
		k, _, ok := parseSSHLine(line)
		if ok && strings.EqualFold(k, key) {
			return i, true
		}
	}
	return -1, false
}

func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	scope := scopeGlobal
	for i, line := range lines {
		scope = updateScope(line, scope)
		if scope == scopeOther {
			continue
		}
		k, _, ok := parseSSHLine(line)
		if ok && strings.EqualFold(k, key) {
			lines[i] = replaceSSHValue(line, value)
			return []byte(strings.Join(lines, "\n")), nil
		}
	}

	// not found — insert into Host * block or create one
	return insertSSHKey(lines, key, value), nil
}

func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	scope := scopeGlobal
	for i, line := range lines {
		scope = updateScope(line, scope)
		if scope == scopeOther {
			continue
		}
		k, _, ok := parseSSHLine(line)
		if ok && strings.EqualFold(k, key) {
			lines = append(lines[:i], lines[i+1:]...)
			return []byte(strings.Join(lines, "\n")), nil
		}
	}
	return data, nil
}

func (p *parser) ListKeys(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	var keys []string
	scope := scopeGlobal
	for _, line := range lines {
		scope = updateScope(line, scope)
		if scope == scopeOther {
			continue
		}
		k, _, ok := parseSSHLine(line)
		if ok && !strings.EqualFold(k, "Host") && !strings.EqualFold(k, "Match") {
			keys = append(keys, k)
		}
	}
	return keys
}

// scope tracking for Host blocks
type sshScope int

const (
	scopeGlobal sshScope = iota // before any Host directive
	scopeWild                   // inside Host * block
	scopeOther                  // inside a specific Host block (skip)
)

func updateScope(line string, current sshScope) sshScope {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed[0] == '#' {
		return current
	}
	fields := strings.Fields(trimmed)
	if len(fields) < 2 {
		return current
	}
	if strings.EqualFold(fields[0], "Host") {
		if fields[1] == "*" {
			return scopeWild
		}
		return scopeOther
	}
	if strings.EqualFold(fields[0], "Match") {
		return scopeOther
	}
	return current
}

// parseSSHLine extracts keyword and value from an SSH config line.
func parseSSHLine(line string) (key, value string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed[0] == '#' {
		return "", "", false
	}

	// handle = separator
	if eqIdx := strings.IndexByte(trimmed, '='); eqIdx >= 0 {
		k := strings.TrimSpace(trimmed[:eqIdx])
		if k != "" && !strings.ContainsAny(k, " \t") {
			v := strings.TrimSpace(trimmed[eqIdx+1:])
			return k, v, true
		}
	}

	// space/tab separator
	idx := strings.IndexAny(trimmed, " \t")
	if idx < 0 {
		return trimmed, "", true
	}

	key = trimmed[:idx]
	value = strings.TrimSpace(trimmed[idx+1:])
	return key, value, true
}

// replaceSSHValue replaces the value portion of an SSH config line,
// preserving the keyword and indentation.
func replaceSSHValue(line, newValue string) string {
	trimmed := strings.TrimSpace(line)
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

	// handle = separator
	if eqIdx := strings.IndexByte(trimmed, '='); eqIdx >= 0 {
		k := strings.TrimSpace(trimmed[:eqIdx])
		if k != "" && !strings.ContainsAny(k, " \t") {
			return indent + k + " = " + newValue
		}
	}

	// space separator
	idx := strings.IndexAny(trimmed, " \t")
	if idx < 0 {
		return line
	}
	keyword := trimmed[:idx]
	return indent + keyword + " " + newValue
}

// insertSSHKey adds a key-value pair into a Host * block, creating it if needed.
func insertSSHKey(lines []string, key, value string) []byte {
	// find Host * block and insert at end of it
	inWild := false
	insertAt := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed[0] == '#' {
			if inWild {
				insertAt = i
			}
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 2 && strings.EqualFold(fields[0], "Host") {
			if inWild {
				// end of Host * block — insert before this line
				result := make([]string, 0, len(lines)+1)
				result = append(result, lines[:i]...)
				result = append(result, "    "+key+" "+value)
				result = append(result, lines[i:]...)
				return []byte(strings.Join(result, "\n"))
			}
			if fields[1] == "*" {
				inWild = true
				insertAt = i
			}
		} else if inWild {
			insertAt = i
		}
	}

	if inWild && insertAt >= 0 {
		// at end of file within Host * block
		result := make([]string, 0, len(lines)+1)
		result = append(result, lines[:insertAt+1]...)
		result = append(result, "    "+key+" "+value)
		result = append(result, lines[insertAt+1:]...)
		return []byte(strings.Join(result, "\n"))
	}

	// no Host * block — add at the top (global scope)
	result := key + " " + value + "\n" + strings.Join(lines, "\n")
	return []byte(result)
}
