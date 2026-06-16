package tmux

import (
	"strings"

	cfgparse "github.com/getkonfi/konfi/pkg/parser"
)

// newParser handles the tmux config format: `set -g key value`,
// `set-option -g key value`, `setw -g key value`. it reuses FlatParser with a
// tmux-aware splitter; new keys are written as global `set -g key value` lines.
type parser struct {
	base cfgparse.FlatParser
}

func newParser() *parser {
	return &parser{base: cfgparse.FlatParser{Split: parseTmuxSet, Format: formatTmuxSet}}
}

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	return p.base.FindValue(data, key)
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	return p.base.FindLine(data, key)
}

func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		k, _, start, end, ok := parseTmuxSetParts(line)
		if !ok || k != key {
			continue
		}
		if start < end {
			lines[i] = line[:start] + value + line[end:]
		} else {
			lines[i] = strings.TrimRight(line[:start], " \t") + " " + value + line[start:]
		}
		return []byte(strings.Join(lines, "\n")), nil
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		lines = append(lines, formatTmuxSet(key, value))
	} else {
		lines = append(lines[:len(lines)-1], formatTmuxSet(key, value), "")
	}
	return []byte(strings.Join(lines, "\n")), nil
}

func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	return p.base.DeleteKey(data, key)
}

func (p *parser) ListKeys(data []byte) []string {
	return p.base.ListKeys(data)
}

func (p *parser) FindAll(data []byte) map[string]string {
	return p.base.FindAll(data)
}

// formatTmuxSet renders a new key as a global set-option line.
func formatTmuxSet(key, value string) string {
	return "set -g " + key + " " + value
}

// parseTmuxSet extracts key and value from a tmux set line.
// handles: set -g key value, set-option -g key value, setw -g key value
func parseTmuxSet(line string) (key, value string, ok bool) {
	key, value, _, _, ok = parseTmuxSetParts(line)
	return key, value, ok
}

type tmuxToken struct {
	text       string
	start, end int
}

func parseTmuxSetParts(line string) (key, value string, valueStart, valueEnd int, ok bool) {
	tokens := tmuxTokens(line)
	if len(tokens) < 3 {
		return "", "", 0, 0, false
	}

	cmd := tokens[0].text
	if cmd != "set" && cmd != "set-option" && cmd != "set-window-option" && cmd != "setw" {
		return "", "", 0, 0, false
	}

	// skip flags (tokens starting with -)
	i := 1
	for i < len(tokens) && strings.HasPrefix(tokens[i].text, "-") {
		i++
	}

	if i >= len(tokens) {
		return "", "", 0, 0, false
	}

	key = tokens[i].text
	i++

	if i < len(tokens) {
		valueStart = tokens[i].start
		valueEnd = tokens[len(tokens)-1].end
		value = strings.TrimSpace(line[valueStart:valueEnd])
	} else {
		valueStart = tokens[i-1].end
		valueEnd = valueStart
	}

	return key, value, valueStart, valueEnd, true
}

func tmuxTokens(line string) []tmuxToken {
	var tokens []tmuxToken
	for i := 0; i < len(line); {
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		if i >= len(line) || startsTmuxComment(line, i) {
			break
		}

		start := i
		var quote byte
		for i < len(line) {
			ch := line[i]
			if quote != 0 {
				if ch == '\\' && quote == '"' && i+1 < len(line) {
					i += 2
					continue
				}
				if ch == quote {
					quote = 0
				}
				i++
				continue
			}
			switch ch {
			case '"', '\'':
				quote = ch
				i++
			case ' ', '\t':
				goto tokenDone
			case '#':
				if startsTmuxComment(line, i) {
					goto tokenDone
				}
				i++
			default:
				i++
			}
		}

	tokenDone:
		if start < i {
			tokens = append(tokens, tmuxToken{text: line[start:i], start: start, end: i})
		}
	}
	return tokens
}

func startsTmuxComment(line string, idx int) bool {
	if line[idx] != '#' {
		return false
	}
	if idx > 0 && line[idx-1] != ' ' && line[idx-1] != '\t' {
		return false
	}
	next := idx + 1
	return next >= len(line) || line[next] == ' ' || line[next] == '\t'
}
