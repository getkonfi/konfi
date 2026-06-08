package powerlevel10k

import "strings"

type parser struct{}

type assignment struct {
	key        string
	line       int
	valueStart int
	valueEnd   int
	array      bool
}

func newParser() *parser {
	return &parser{}
}

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		a, ok := parseAssignment(lines[i], i)
		if !ok || a.key != key {
			continue
		}
		if a.array {
			vals, _, ok := parseArray(lines, i, a.valueStart)
			if !ok {
				return "", false
			}
			return strings.Join(vals, ", "), true
		}
		return unquoteZsh(strings.TrimSpace(lines[i][a.valueStart:a.valueEnd])), true
	}
	return "", false
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		a, ok := parseAssignment(line, i)
		if ok && a.key == key {
			return i, true
		}
	}
	return -1, false
}

func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		a, ok := parseAssignment(lines[i], i)
		if !ok || a.key != key {
			continue
		}
		if a.array {
			end := i
			if _, e, ok := parseArray(lines, i, a.valueStart); ok {
				end = e
			}
			repl := []string{linePrefix(lines[i], a.valueStart) + value}
			lines = replaceLines(lines, i, end, repl)
			return []byte(strings.Join(lines, "\n")), nil
		}
		lines[i] = lines[i][:a.valueStart] + value + lines[i][a.valueEnd:]
		return []byte(strings.Join(lines, "\n")), nil
	}

	lines = appendConfigLines(data, lines, []string{formatAssignment(key, value)})
	return []byte(strings.Join(lines, "\n")), nil
}

func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		a, ok := parseAssignment(lines[i], i)
		if !ok || a.key != key {
			out = append(out, lines[i])
			continue
		}
		if a.array {
			if _, end, ok := parseArray(lines, i, a.valueStart); ok {
				i = end
			}
		}
	}
	return []byte(strings.Join(out, "\n")), nil
}

func (p *parser) ListKeys(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	var keys []string
	for i := 0; i < len(lines); i++ {
		a, ok := parseAssignment(lines[i], i)
		if !ok {
			continue
		}
		keys = append(keys, a.key)
		if a.array {
			if _, end, ok := parseArray(lines, i, a.valueStart); ok {
				i = end
			}
		}
	}
	return keys
}

func (p *parser) FindValues(data []byte, key string) ([]string, bool) {
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		a, ok := parseAssignment(lines[i], i)
		if !ok || a.key != key {
			continue
		}
		if a.array {
			vals, _, ok := parseArray(lines, i, a.valueStart)
			return vals, ok
		}
		return []string{unquoteZsh(strings.TrimSpace(lines[i][a.valueStart:a.valueEnd]))}, true
	}
	return nil, false
}

func (p *parser) SetValues(data []byte, key string, values []string) ([]byte, error) {
	values = cleanValues(values)
	if len(values) == 0 {
		return p.DeleteKey(data, key)
	}

	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		a, ok := parseAssignment(lines[i], i)
		if !ok || a.key != key {
			continue
		}
		end := i
		if a.array {
			if _, e, ok := parseArray(lines, i, a.valueStart); ok {
				end = e
			}
		}
		block := formatArrayBlock(linePrefix(lines[i], a.valueStart), leadingIndent(lines[i]), values)
		lines = replaceLines(lines, i, end, block)
		return []byte(strings.Join(lines, "\n")), nil
	}

	lines = appendConfigLines(data, lines, formatArrayBlock("typeset -g "+key+"=", "", values))
	return []byte(strings.Join(lines, "\n")), nil
}

func (p *parser) FindAll(data []byte) map[string]string {
	singles, multi := p.FindAllMulti(data)
	for key, vals := range multi {
		singles[key] = strings.Join(vals, ", ")
	}
	return singles
}

func (p *parser) FindAllMulti(data []byte) (singles map[string]string, multi map[string][]string) {
	lines := strings.Split(string(data), "\n")
	singles = make(map[string]string, len(lines)/2)
	multi = make(map[string][]string)
	for i := 0; i < len(lines); i++ {
		a, ok := parseAssignment(lines[i], i)
		if !ok {
			continue
		}
		if a.array {
			vals, end, ok := parseArray(lines, i, a.valueStart)
			if ok {
				multi[a.key] = vals
				i = end
			}
			continue
		}
		singles[a.key] = unquoteZsh(strings.TrimSpace(lines[i][a.valueStart:a.valueEnd]))
	}
	return singles, multi
}

func parseAssignment(line string, lineIdx int) (assignment, bool) {
	if isBlankOrComment(line) {
		return assignment{}, false
	}
	offset := len(line) - len(strings.TrimLeft(line, " \t"))
	body := line[offset:]
	eqRel := strings.IndexByte(body, '=')
	if eqRel < 0 {
		return assignment{}, false
	}
	left := body[:eqRel]
	fields := strings.Fields(left)
	if len(fields) == 0 {
		return assignment{}, false
	}
	key := fields[len(fields)-1]
	if !isConfigKey(key) {
		return assignment{}, false
	}
	if len(fields) > 1 && !isAssignmentCommand(fields[0]) {
		return assignment{}, false
	}

	valueStart := offset + eqRel + 1
	tail := line[valueStart:]
	valueEndRel := shellValueEnd(tail)
	valueText := strings.TrimSpace(tail[:valueEndRel])
	return assignment{
		key:        key,
		line:       lineIdx,
		valueStart: valueStart,
		valueEnd:   valueStart + len(strings.TrimRight(tail[:valueEndRel], " \t")),
		array:      strings.HasPrefix(valueText, "("),
	}, true
}

func parseArray(lines []string, startLine, startCol int) ([]string, int, bool) {
	var vals []string
	var tok strings.Builder
	started := false
	inSingle := false
	inDouble := false
	escaped := false

	flush := func() {
		if tok.Len() == 0 {
			return
		}
		vals = append(vals, tok.String())
		tok.Reset()
	}

	for lineIdx := startLine; lineIdx < len(lines); lineIdx++ {
		line := lines[lineIdx]
		col := 0
		if lineIdx == startLine {
			col = startCol
		}
		for i := col; i < len(line); i++ {
			ch := line[i]
			if !started {
				if isSpace(ch) {
					continue
				}
				if ch != '(' {
					return nil, lineIdx, false
				}
				started = true
				continue
			}

			if inSingle {
				if ch == '\'' {
					inSingle = false
				} else {
					tok.WriteByte(ch)
				}
				continue
			}
			if inDouble {
				switch {
				case escaped:
					tok.WriteByte(ch)
					escaped = false
				case ch == '\\':
					escaped = true
				case ch == '"':
					inDouble = false
				default:
					tok.WriteByte(ch)
				}
				continue
			}

			switch {
			case ch == '\'':
				inSingle = true
			case ch == '"':
				inDouble = true
			case ch == '#':
				flush()
				i = len(line)
			case ch == ')':
				flush()
				return vals, lineIdx, true
			case isSpace(ch):
				flush()
			default:
				tok.WriteByte(ch)
			}
		}
		if started && !inSingle && !inDouble {
			flush()
		}
	}
	return nil, len(lines) - 1, false
}

func shellValueEnd(s string) int {
	inSingle := false
	inDouble := false
	escaped := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inSingle {
			if ch == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			switch {
			case escaped:
				escaped = false
			case ch == '\\':
				escaped = true
			case ch == '"':
				inDouble = false
			}
			continue
		}
		switch ch {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '#':
			if i == 0 || isSpace(s[i-1]) {
				return i
			}
		}
	}
	return len(s)
}

func unquoteZsh(value string) string {
	if len(value) < 2 {
		return value
	}
	if value[0] == '\'' && value[len(value)-1] == '\'' {
		return strings.ReplaceAll(value[1:len(value)-1], "'\\''", "'")
	}
	if value[0] == '"' && value[len(value)-1] == '"' {
		var out strings.Builder
		escaped := false
		for i := 1; i < len(value)-1; i++ {
			ch := value[i]
			if escaped {
				out.WriteByte(ch)
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			out.WriteByte(ch)
		}
		if escaped {
			out.WriteByte('\\')
		}
		return out.String()
	}
	return value
}

func formatAssignment(key, value string) string {
	return "typeset -g " + key + "=" + value
}

func formatArrayBlock(prefix, indent string, values []string) []string {
	block := []string{prefix + "("}
	for _, value := range values {
		block = append(block, indent+"  "+formatArrayValue(value))
	}
	block = append(block, indent+")")
	return block
}

func formatArrayValue(value string) string {
	if value == "" || !isBareArrayWord(value) {
		return quoteZsh(value)
	}
	return value
}

func quoteZsh(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func cleanValues(values []string) []string {
	out := values[:0]
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func appendConfigLines(data []byte, lines []string, add []string) []string {
	if len(lines) == 1 && lines[0] == "" && len(data) == 0 {
		return add
	}
	if len(data) > 0 && data[len(data)-1] == '\n' {
		out := append([]string{}, lines[:len(lines)-1]...)
		out = append(out, add...)
		return append(out, "")
	}
	return append(lines, add...)
}

func replaceLines(lines []string, start, end int, repl []string) []string {
	out := append([]string{}, lines[:start]...)
	out = append(out, repl...)
	return append(out, lines[end+1:]...)
}

func linePrefix(line string, valueStart int) string {
	if valueStart < 0 || valueStart > len(line) {
		return ""
	}
	return line[:valueStart]
}

func leadingIndent(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

func isBlankOrComment(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed == "" || strings.HasPrefix(trimmed, "#")
}

func isAssignmentCommand(cmd string) bool {
	switch cmd {
	case "typeset", "local", "readonly", "export":
		return true
	default:
		return false
	}
}

func isConfigKey(key string) bool {
	return strings.HasPrefix(key, "POWERLEVEL9K_") ||
		key == "DEFAULT_USER" ||
		key == "ZLE_RPROMPT_INDENT"
}

func isBareArrayWord(value string) bool {
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_' || ch == '-' ||
			ch == '.' || ch == '/' || ch == ':' {
			continue
		}
		return false
	}
	return true
}

func isSpace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\r'
}
