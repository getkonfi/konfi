package yazi

import (
	"strconv"
	"strings"

	parserpkg "github.com/getkonfi/konfi/pkg/parser"
)

type yaziParser struct {
	base *parserpkg.SectionParser
}

func newParser() *yaziParser {
	return &yaziParser{base: &parserpkg.SectionParser{SplitKey: parserpkg.SplitKeyLast}}
}

func (p *yaziParser) FindValue(data []byte, key string) (string, bool) {
	configKey := configKey(key)
	if isRawTOMLKey(configKey) {
		if value, _, ok := p.findRawTOMLValue(data, configKey); ok {
			return value, true
		}
	}
	if value, ok := p.base.FindValue(data, configKey); ok {
		return value, true
	}
	if configKey != key {
		return p.base.FindValue(data, key)
	}
	return "", false
}

func (p *yaziParser) FindLine(data []byte, key string) (int, bool) {
	configKey := configKey(key)
	if line, ok := p.base.FindLine(data, configKey); ok {
		return line, true
	}
	if configKey != key {
		return p.base.FindLine(data, key)
	}
	return -1, false
}

func (p *yaziParser) SetValue(data []byte, key, value string) ([]byte, error) {
	configKey := configKey(key)
	if isRawTOMLKey(configKey) {
		value = rawTOMLWriteValue(value)
		if _, start, end, ok := p.findRawTOMLSpan(data, configKey); ok {
			return replaceRawSpan(data, start, end, value), nil
		}
	}
	if configKey != key {
		if _, ok := p.base.FindLine(data, key); ok {
			return p.base.SetValue(data, key, value)
		}
	}
	return p.base.SetValue(data, configKey, value)
}

func (p *yaziParser) DeleteKey(data []byte, key string) ([]byte, error) {
	configKey := configKey(key)
	if isRawTOMLKey(configKey) {
		if _, start, end, ok := p.findRawTOMLSpan(data, configKey); ok {
			return deleteRawSpan(data, start, end), nil
		}
	}
	if _, ok := p.base.FindLine(data, configKey); ok {
		return p.base.DeleteKey(data, configKey)
	}
	if configKey != key {
		return p.base.DeleteKey(data, key)
	}
	return p.base.DeleteKey(data, configKey)
}

func (p *yaziParser) ListKeys(data []byte) []string {
	keys := p.base.ListKeys(p.keyScanData(data))
	for i, key := range keys {
		keys[i] = schemaKey(key)
	}
	return keys
}

func (p *yaziParser) FindAll(data []byte) map[string]string {
	base := p.base.FindAll(p.keyScanData(data))
	out := make(map[string]string, len(base))
	for key, value := range base {
		out[schemaKey(key)] = value
	}
	for _, key := range rawTOMLKeys {
		if value, ok := p.FindValue(data, key); ok {
			out[key] = value
		}
	}
	return out
}

func configKey(key string) string {
	if rest, ok := strings.CutPrefix(key, "manager."); ok {
		return "mgr." + rest
	}
	return key
}

func schemaKey(key string) string {
	if rest, ok := strings.CutPrefix(key, "mgr."); ok {
		return "manager." + rest
	}
	return key
}

func isRawTOMLKey(key string) bool {
	for _, rawKey := range rawTOMLKeys {
		if key == rawKey {
			return true
		}
	}
	return false
}

var rawTOMLKeys = []string{"opener.edit", "open.rules"}

func rawTOMLWriteValue(value string) string {
	unquoted, err := strconv.Unquote(value)
	if err != nil {
		return value
	}
	trimmed := strings.TrimSpace(unquoted)
	if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
		return unquoted
	}
	return value
}

func (p *yaziParser) findRawTOMLValue(data []byte, key string) (value string, start int, ok bool) {
	value, start, _, ok = p.findRawTOMLSpan(data, key)
	return value, start, ok
}

func (p *yaziParser) findRawTOMLSpan(data []byte, key string) (value string, start, end int, ok bool) {
	line, ok := p.base.FindLine(data, key)
	if !ok {
		return "", -1, -1, false
	}
	lines := strings.Split(string(data), "\n")
	if line < 0 || line >= len(lines) {
		return "", -1, -1, false
	}
	eq := strings.IndexByte(lines[line], '=')
	if eq < 0 {
		return "", -1, -1, false
	}
	value, end, ok = scanRawTOML(lines, line, eq+1)
	if !ok {
		return "", -1, -1, false
	}
	return value, line, end, true
}

func (p *yaziParser) keyScanData(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	for _, key := range rawTOMLKeys {
		_, start, end, ok := p.findRawTOMLSpan(data, key)
		if !ok {
			continue
		}
		if start < 0 || end < start || end >= len(lines) {
			continue
		}
		eq := strings.IndexByte(lines[start], '=')
		if eq < 0 {
			continue
		}
		prefix := lines[start][:eq+1]
		if rest := lines[start][eq+1:]; rest != "" && rest[0] == ' ' {
			prefix += " "
		}
		lines[start] = prefix + "[]"
		for i := start + 1; i <= end; i++ {
			lines[i] = ""
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

type tomlScanState struct {
	quote   byte
	escaped bool
}

func scanRawTOML(lines []string, start, valueStart int) (value string, end int, ok bool) {
	var b strings.Builder
	var state tomlScanState
	balance := 0
	seen := false
	for i := start; i < len(lines); i++ {
		part := lines[i]
		if i == start {
			part = part[valueStart:]
		}
		if i > start {
			b.WriteByte('\n')
		}
		b.WriteString(part)
		delta, lineSeen := rawTOMLBracketDelta(part, &state)
		balance += delta
		seen = seen || lineSeen
		if seen && balance <= 0 && state.quote == 0 {
			return strings.TrimSpace(b.String()), i, true
		}
	}
	if seen {
		return strings.TrimSpace(b.String()), len(lines) - 1, true
	}
	return "", -1, false
}

func rawTOMLBracketDelta(s string, state *tomlScanState) (int, bool) {
	delta := 0
	seen := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if state.quote != 0 {
			if state.quote == '"' && state.escaped {
				state.escaped = false
				continue
			}
			if state.quote == '"' && ch == '\\' {
				state.escaped = true
				continue
			}
			if ch == state.quote {
				state.quote = 0
			}
			continue
		}
		switch ch {
		case '#':
			return delta, seen
		case '"', '\'':
			state.quote = ch
		case '[', '{':
			delta++
			seen = true
		case ']', '}':
			delta--
			seen = true
		}
	}
	return delta, seen
}

func replaceRawSpan(data []byte, start, end int, value string) []byte {
	lines := strings.Split(string(data), "\n")
	if start < 0 || end < start || end >= len(lines) {
		return data
	}
	eq := strings.IndexByte(lines[start], '=')
	if eq < 0 {
		return data
	}
	prefix := lines[start][:eq+1]
	if rest := lines[start][eq+1:]; rest != "" && rest[0] == ' ' {
		prefix += " "
	}
	valueLines := strings.Split(value, "\n")
	replacement := make([]string, len(valueLines))
	replacement[0] = prefix + valueLines[0]
	copy(replacement[1:], valueLines[1:])

	out := make([]string, 0, len(lines)-(end-start)+len(replacement)-1)
	out = append(out, lines[:start]...)
	out = append(out, replacement...)
	out = append(out, lines[end+1:]...)
	return []byte(strings.Join(out, "\n"))
}

func deleteRawSpan(data []byte, start, end int) []byte {
	lines := strings.Split(string(data), "\n")
	if start < 0 || end < start || end >= len(lines) {
		return data
	}
	out := make([]string, 0, len(lines)-(end-start+1))
	out = append(out, lines[:start]...)
	out = append(out, lines[end+1:]...)
	return []byte(strings.Join(out, "\n"))
}
