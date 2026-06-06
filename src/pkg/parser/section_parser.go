package parser

import (
	"strconv"
	"strings"
)

// scanMultiline checks whether the assignment on lines[i] opens a TOML
// multi-line string (""" basic or ''' literal). if so it consumes through the
// closing delimiter and returns the decoded logical value plus the inclusive
// end line. ok=false means lines[i] is an ordinary single-line assignment and
// the caller should parse it the normal way — so single-line behavior is
// untouched and only triple-quote values trigger this path.
func scanMultiline(lines []string, i int) (value string, end int, ok bool) {
	if i < 0 || i >= len(lines) {
		return "", i, false
	}
	eq := strings.IndexByte(lines[i], '=')
	if eq < 0 {
		return "", i, false
	}
	raw := strings.TrimSpace(lines[i][eq+1:])
	var delim string
	switch {
	case strings.HasPrefix(raw, `"""`):
		delim = `"""`
	case strings.HasPrefix(raw, "'''"):
		delim = "'''"
	default:
		return "", i, false
	}

	rest := raw[len(delim):]
	// closes on the same line: """x"""
	if idx := strings.Index(rest, delim); idx >= 0 {
		return decodeTOMLString(delim, rest[:idx]), i, true
	}
	// spans multiple lines: accumulate until the closing delimiter
	var sb strings.Builder
	sb.WriteString(rest)
	sb.WriteByte('\n')
	for j := i + 1; j < len(lines); j++ {
		if idx := strings.Index(lines[j], delim); idx >= 0 {
			sb.WriteString(lines[j][:idx])
			return decodeTOMLString(delim, sb.String()), j, true
		}
		sb.WriteString(lines[j])
		sb.WriteByte('\n')
	}
	// unterminated — treat the remainder as the value
	return decodeTOMLString(delim, sb.String()), len(lines) - 1, true
}

// decodeTOMLString applies multi-line TOML string semantics to the raw inner
// content. a newline immediately after the opening delimiter is trimmed;
// literal ('''') strings are verbatim; basic (""") strings honor escapes and
// line-ending backslash continuations.
func decodeTOMLString(delim, s string) string {
	s = strings.TrimPrefix(s, "\n")
	if delim == "'''" {
		return s
	}
	return decodeBasicMultiline(s)
}

func decodeBasicMultiline(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' || i+1 >= len(s) {
			b.WriteByte(c)
			continue
		}
		n := s[i+1]
		// line-ending backslash: trims the newline and all leading whitespace
		// up to the next non-whitespace character
		if n == '\n' || n == '\r' || n == ' ' || n == '\t' {
			j := i + 1
			for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\r') {
				j++
			}
			if j < len(s) && s[j] == '\n' {
				j++
				for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\r' || s[j] == '\n') {
					j++
				}
				i = j - 1
				continue
			}
		}
		switch n {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case '"':
			b.WriteByte('"')
		case '\\':
			b.WriteByte('\\')
		case 'u', 'U':
			width := 4
			if n == 'U' {
				width = 8
			}
			if i+2+width <= len(s) {
				if r, err := strconv.ParseInt(s[i+2:i+2+width], 16, 32); err == nil {
					b.WriteRune(rune(r))
					i += 1 + width
					continue
				}
			}
			b.WriteByte('\\')
			b.WriteByte(n)
		default:
			b.WriteByte('\\')
			b.WriteByte(n)
		}
		i++
	}
	return b.String()
}

// replaceSpan replaces physical lines start..end (inclusive) with a single
// "key = value" line, preserving the key and spacing from the start line.
// used to collapse an edited multi-line string back to one line.
func replaceSpan(data []byte, start, end int, newValue string) []byte {
	lines := strings.Split(string(data), "\n")
	if start < 0 || end < start || end >= len(lines) {
		return data
	}
	eqIdx := strings.IndexByte(lines[start], '=')
	if eqIdx < 0 {
		return data
	}
	prefix := lines[start][:eqIdx+1]
	if rest := lines[start][eqIdx+1:]; rest != "" && rest[0] == ' ' {
		prefix += " "
	}
	out := make([]string, 0, len(lines)-(end-start))
	out = append(out, lines[:start]...)
	out = append(out, prefix+newValue)
	out = append(out, lines[end+1:]...)
	return []byte(strings.Join(out, "\n"))
}

// KeySplitter splits a dotted config key into section and field parts.
type KeySplitter func(key string) (section, field string)

// SplitKeyLast splits at the last dot: "a.b.c" → ("a.b", "c").
// use for TOML configs with dotted section headers (alacritty, helix, rio).
func SplitKeyLast(key string) (section, field string) {
	idx := strings.LastIndexByte(key, '.')
	if idx < 0 {
		return "", key
	}
	return key[:idx], key[idx+1:]
}

// SplitKeyFirst splits at the first dot: "a.b" → ("a", "b").
// use for INI-style configs with flat section names (git, starship, pacman).
func SplitKeyFirst(key string) (section, field string) {
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
	var val string
	var lineIdx int
	var found bool
	if section == "" {
		val, lineIdx, found = FindTopLevelKey(data, field)
	} else {
		val, lineIdx, found = FindKeyInSection(data, section, field)
	}
	if !found {
		return "", false
	}
	// decode the full block when the located line opens a multi-line string
	if mv, _, ok := scanMultiline(strings.Split(string(data), "\n"), lineIdx); ok {
		return mv, true
	}
	return val, true
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
			return p.replaceValue(data, lineIdx, value), nil
		}
		return InsertTopLevelKey(data, field, value), nil
	}

	_, lineIdx, found := FindKeyInSection(data, section, field)
	if found {
		return p.replaceValue(data, lineIdx, value), nil
	}
	return InsertKeyInSection(data, section, field, value), nil
}

// replaceValue rewrites the value at lineIdx, collapsing a multi-line string
// block onto a single line when the located key opens one.
func (p *SectionParser) replaceValue(data []byte, lineIdx int, value string) []byte {
	if _, end, ok := scanMultiline(strings.Split(string(data), "\n"), lineIdx); ok && end > lineIdx {
		return replaceSpan(data, lineIdx, end, value)
	}
	return ReplaceValueOnLine(data, lineIdx, value)
}

func (p *SectionParser) DeleteKey(data []byte, key string) ([]byte, error) {
	section, field := p.SplitKey(key)

	if section == "" {
		_, lineIdx, found := FindTopLevelKey(data, field)
		if !found {
			return data, nil
		}
		return p.deleteAt(data, lineIdx), nil
	}

	_, lineIdx, found := FindKeyInSection(data, section, field)
	if !found {
		return data, nil
	}
	return p.deleteAt(data, lineIdx), nil
}

// deleteAt removes the assignment at lineIdx, including the full block when the
// key opens a multi-line string.
func (p *SectionParser) deleteAt(data []byte, lineIdx int) []byte {
	if _, end, ok := scanMultiline(strings.Split(string(data), "\n"), lineIdx); ok && end > lineIdx {
		lines := strings.Split(string(data), "\n")
		out := append(lines[:lineIdx], lines[end+1:]...)
		return []byte(strings.Join(out, "\n"))
	}
	return DeleteKeyOnLine(data, lineIdx)
}

// FindAll returns all key-value pairs in a single pass, using dotted keys.
func (p *SectionParser) FindAll(data []byte) map[string]string {
	lines := strings.Split(string(data), "\n")
	m := make(map[string]string)
	currentSection := ""
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
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
		// decode + skip the body of a multi-line string value
		if mv, end, isMulti := scanMultiline(lines, i); isMulti {
			v = mv
			i = end
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
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
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
		// skip the body lines of a multi-line string value
		if _, end, isMulti := scanMultiline(lines, i); isMulti {
			i = end
		}
		if currentSection != "" {
			keys = append(keys, currentSection+"."+k)
		} else {
			keys = append(keys, k)
		}
	}
	return keys
}
