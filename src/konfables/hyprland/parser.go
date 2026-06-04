package hyprland

import (
	"bytes"
	"fmt"
	"strings"
)

type parser struct{}

// newParser returns a hyprland config parser.
func newParser() *parser { return &parser{} }

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	v, _, found := findResult(data, key)
	return v, found
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	_, i, found := findResult(data, key)
	return i, found
}

func findResult(data []byte, key string) (value string, lineIdx int, found bool) {
	block, inner, nested := splitKey(key)
	if !nested {
		return findFlatResult(data, key)
	}
	return findNestedResult(data, block, inner)
}

func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	block, inner, nested := splitKey(key)
	if !nested {
		return setFlat(data, key, value)
	}
	return setNested(data, block, inner, value)
}

func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	block, inner, nested := splitKey(key)
	if !nested {
		return deleteFlat(data, key)
	}
	return deleteNested(data, block, inner)
}

// FindAll returns all key-value pairs as dotted paths in a single pass.
func (p *parser) FindAll(data []byte) map[string]string {
	lines := bytes.Split(data, []byte("\n"))
	m := make(map[string]string)
	var stack []string

	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0] == '#' {
			continue
		}

		s := string(trimmed)
		if bytes.HasSuffix(trimmed, []byte("{")) {
			name := strings.TrimSpace(strings.TrimSuffix(s, "{"))
			if name != "" {
				stack = append(stack, name)
			}
			continue
		}

		if bytes.Equal(trimmed, []byte("}")) {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			continue
		}

		if k, v, ok := parseLine(trimmed); ok {
			if len(stack) > 0 {
				m[strings.Join(stack, ".")+"."+k] = v
			} else {
				m[k] = v
			}
		}
	}
	return m
}

// ListKeys returns all config keys defined in the data as dotted paths.
func (p *parser) ListKeys(data []byte) []string {
	lines := bytes.Split(data, []byte("\n"))
	var keys []string
	var stack []string

	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0] == '#' {
			continue
		}

		// check for block open: "name {" or "name{"
		s := string(trimmed)
		if bytes.HasSuffix(trimmed, []byte("{")) {
			name := strings.TrimSpace(strings.TrimSuffix(s, "{"))
			if name != "" {
				stack = append(stack, name)
			}
			continue
		}

		// closing brace
		if bytes.Equal(trimmed, []byte("}")) {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			continue
		}

		// key = value line
		if k, _, ok := parseLine(trimmed); ok {
			if len(stack) > 0 {
				keys = append(keys, strings.Join(stack, ".")+"."+k)
			} else {
				keys = append(keys, k)
			}
		}
	}

	return keys
}

// splitKey splits a dotted key into block and inner key.
// returns (block, inner, isNested).
func splitKey(key string) (block, inner string, nested bool) {
	idx := strings.IndexByte(key, '.')
	if idx < 0 {
		return "", "", false
	}
	return key[:idx], key[idx+1:], true
}

// findFlatResult finds a key = value at depth 0, returning value, line index, and found.
func findFlatResult(data []byte, key string) (value string, lineIdx int, found bool) {
	lines := bytes.Split(data, []byte("\n"))
	depth := 0
	for i, line := range lines {
		trimmed := bytes.TrimSpace(line)
		depth += braceBalance(trimmed)
		if depth == 0 {
			if k, v, ok := parseLine(trimmed); ok && k == key {
				return v, i, true
			}
		}
	}
	return "", -1, false
}

// findNestedResult finds a key inside a named block, returning value, line index, and found.
// handles depth-2+ by recursing into sub-blocks with offset tracking.
func findNestedResult(data []byte, block, inner string) (value string, lineIdx int, found bool) {
	if v, i, ok := findDirectNestedResult(data, block, inner); ok {
		return v, i, true
	}
	if subBlock, subInner, nested := splitKey(inner); nested {
		return findNestedRecursive(data, block, subBlock, subInner)
	}
	return "", -1, false
}

func findDirectNestedResult(data []byte, block, inner string) (value string, lineIdx int, found bool) {
	lines := bytes.Split(data, []byte("\n"))
	inBlock, depth := false, 0

	for i, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if !inBlock && depth == 0 {
			if isBlockOpen(trimmed, block) {
				inBlock = true
				depth = 1
				continue
			}
		}
		if inBlock {
			depth += braceBalance(trimmed)
			if depth == 1 {
				if k, v, ok := parseLine(trimmed); ok && k == inner {
					return v, i, true
				}
			}
			if depth <= 0 {
				return "", -1, false
			}
		} else {
			depth += braceBalance(trimmed)
		}
	}
	return "", -1, false
}

func findNestedRecursive(data []byte, block, subBlock, subInner string) (value string, lineIdx int, found bool) {
	lines := bytes.Split(data, []byte("\n"))
	inBlock, depth := false, 0
	blockStart, blockEnd := -1, -1

	for i, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if !inBlock && depth == 0 {
			if isBlockOpen(trimmed, block) {
				inBlock = true
				depth = 1
				blockStart = i + 1
				continue
			}
		}
		if inBlock {
			depth += braceBalance(trimmed)
			if depth <= 0 {
				blockEnd = i
				break
			}
		} else {
			depth += braceBalance(trimmed)
		}
	}

	if blockStart < 0 || blockEnd < 0 {
		return "", -1, false
	}

	blockContent := bytes.Join(lines[blockStart:blockEnd], []byte("\n"))
	v, lineInBlock, found := findNestedResult(blockContent, subBlock, subInner)
	if !found {
		return "", -1, false
	}
	return v, blockStart + lineInBlock, true
}

// setFlat replaces or appends a flat key = value at depth 0.
func setFlat(data []byte, key, value string) ([]byte, error) {
	lines := bytes.Split(data, []byte("\n"))
	depth := 0
	for i, line := range lines {
		trimmed := bytes.TrimSpace(line)
		depth += braceBalance(trimmed)
		if depth == 0 {
			if k, _, ok := parseLine(trimmed); ok && k == key {
				lines[i] = []byte(key + " = " + value)
				return bytes.Join(lines, []byte("\n")), nil
			}
		}
	}
	// not found — append
	result := ensureTrailingNewline(data)
	result = append(result, []byte(key+" = "+value+"\n")...)
	return result, nil
}

// setNested replaces or inserts a key inside a named block.
// if inner contains dots, recursively descends into sub-blocks.
func setNested(data []byte, block, inner, value string) ([]byte, error) {
	if subBlock, subInner, nested := splitKey(inner); nested {
		if _, _, found := findDirectNestedResult(data, block, inner); found || isLiteralNestedKey(block, inner) {
			return setNestedDirect(data, block, inner, value)
		}
		return setNestedRecursive(data, block, subBlock, subInner, value)
	}
	return setNestedDirect(data, block, inner, value)
}

func setNestedDirect(data []byte, block, inner, value string) ([]byte, error) {
	lines := bytes.Split(data, []byte("\n"))
	inBlock, depth := false, 0
	lastLineInBlock := -1

	for i, line := range lines {
		trimmed := bytes.TrimSpace(line)

		if !inBlock && depth == 0 {
			if isBlockOpen(trimmed, block) {
				inBlock = true
				depth = 1
				continue
			}
		}

		if inBlock {
			depth += braceBalance(trimmed)
			if depth == 1 {
				if k, _, ok := parseLine(trimmed); ok && k == inner {
					// replace in place, preserving leading whitespace
					indent := leadingWhitespace(line)
					lines[i] = []byte(indent + inner + " = " + value)
					return bytes.Join(lines, []byte("\n")), nil
				}
				// track last content line position for insertion
				lastLineInBlock = i
			}
			if depth <= 0 {
				// insert before closing brace
				newLine := []byte("    " + inner + " = " + value)
				insertAt := i
				if lastLineInBlock >= 0 {
					insertAt = lastLineInBlock + 1
				}
				lines = insertLine(lines, insertAt, newLine)
				return bytes.Join(lines, []byte("\n")), nil
			}
		} else {
			depth += braceBalance(trimmed)
		}
	}

	// block not found — create it at end
	result := ensureTrailingNewline(data)
	newBlock := fmt.Sprintf("%s {\n    %s = %s\n}\n", block, inner, value)
	result = append(result, []byte(newBlock)...)
	return result, nil
}

// setNestedRecursive handles setting values in depth-2+ blocks.
// it finds the outer block, extracts its content, applies the nested set,
// then splices the modified content back.
func setNestedRecursive(data []byte, outerBlock, innerBlock, key, value string) ([]byte, error) {
	lines := bytes.Split(data, []byte("\n"))
	inBlock, depth := false, 0
	blockStart, blockEnd := -1, -1

	for i, line := range lines {
		trimmed := bytes.TrimSpace(line)

		if !inBlock && depth == 0 {
			if isBlockOpen(trimmed, outerBlock) {
				inBlock = true
				depth = 1
				blockStart = i + 1
				continue
			}
		}

		if inBlock {
			depth += braceBalance(trimmed)
			if depth <= 0 {
				blockEnd = i
				break
			}
		} else {
			depth += braceBalance(trimmed)
		}
	}

	if blockStart < 0 || blockEnd < 0 {
		// outer block not found — create nested structure
		result := ensureTrailingNewline(data)
		newBlock := fmt.Sprintf("%s {\n    %s {\n        %s = %s\n    }\n}\n",
			outerBlock, innerBlock, key, value)
		result = append(result, []byte(newBlock)...)
		return result, nil
	}

	// extract the block's inner content
	blockContent := bytes.Join(lines[blockStart:blockEnd], []byte("\n"))

	// recursively set value inside the block content
	modified, err := setNested(blockContent, innerBlock, key, value)
	if err != nil {
		return nil, err
	}

	// splice modified content back
	modifiedLines := bytes.Split(modified, []byte("\n"))
	result := make([][]byte, 0, len(lines))
	result = append(result, lines[:blockStart]...)
	result = append(result, modifiedLines...)
	result = append(result, lines[blockEnd:]...)
	return bytes.Join(result, []byte("\n")), nil
}

// deleteFlat removes a flat key line at depth 0.
func deleteFlat(data []byte, key string) ([]byte, error) {
	lines := bytes.Split(data, []byte("\n"))
	depth := 0
	for i, line := range lines {
		trimmed := bytes.TrimSpace(line)
		depth += braceBalance(trimmed)
		if depth == 0 {
			if k, _, ok := parseLine(trimmed); ok && k == key {
				lines = removeLine(lines, i)
				return bytes.Join(lines, []byte("\n")), nil
			}
		}
	}
	return data, nil
}

// deleteNested removes a key line inside a named block.
// if inner contains dots, recursively descends into sub-blocks.
func deleteNested(data []byte, block, inner string) ([]byte, error) {
	if subBlock, subInner, nested := splitKey(inner); nested {
		if _, _, found := findDirectNestedResult(data, block, inner); found || isLiteralNestedKey(block, inner) {
			return deleteNestedDirect(data, block, inner)
		}
		return deleteNestedRecursive(data, block, subBlock, subInner)
	}
	return deleteNestedDirect(data, block, inner)
}

func deleteNestedDirect(data []byte, block, inner string) ([]byte, error) {
	lines := bytes.Split(data, []byte("\n"))
	inBlock, depth := false, 0

	for i, line := range lines {
		trimmed := bytes.TrimSpace(line)

		if !inBlock && depth == 0 {
			if isBlockOpen(trimmed, block) {
				inBlock = true
				depth = 1
				continue
			}
		}

		if inBlock {
			depth += braceBalance(trimmed)
			if depth == 1 {
				if k, _, ok := parseLine(trimmed); ok && k == inner {
					lines = removeLine(lines, i)
					return bytes.Join(lines, []byte("\n")), nil
				}
			}
			if depth <= 0 {
				return data, nil
			}
		} else {
			depth += braceBalance(trimmed)
		}
	}
	return data, nil
}

func isLiteralNestedKey(block, inner string) bool {
	return block == "general" && strings.HasPrefix(inner, "col.")
}

// deleteNestedRecursive handles deleting values in depth-2+ blocks.
func deleteNestedRecursive(data []byte, outerBlock, innerBlock, key string) ([]byte, error) {
	lines := bytes.Split(data, []byte("\n"))
	inBlock, depth := false, 0
	blockStart, blockEnd := -1, -1

	for i, line := range lines {
		trimmed := bytes.TrimSpace(line)

		if !inBlock && depth == 0 {
			if isBlockOpen(trimmed, outerBlock) {
				inBlock = true
				depth = 1
				blockStart = i + 1
				continue
			}
		}

		if inBlock {
			depth += braceBalance(trimmed)
			if depth <= 0 {
				blockEnd = i
				break
			}
		} else {
			depth += braceBalance(trimmed)
		}
	}

	if blockStart < 0 || blockEnd < 0 {
		return data, nil
	}

	blockContent := bytes.Join(lines[blockStart:blockEnd], []byte("\n"))
	modified, err := deleteNested(blockContent, innerBlock, key)
	if err != nil {
		return nil, err
	}

	modifiedLines := bytes.Split(modified, []byte("\n"))
	result := make([][]byte, 0, len(lines))
	result = append(result, lines[:blockStart]...)
	result = append(result, modifiedLines...)
	result = append(result, lines[blockEnd:]...)
	return bytes.Join(result, []byte("\n")), nil
}

// parseLine extracts key and value from "key = value" (including $variable lines).
// returns ("", "", false) for comments, blank lines, or block openers.
func parseLine(trimmed []byte) (key, value string, ok bool) {
	if len(trimmed) == 0 || trimmed[0] == '#' {
		return "", "", false
	}
	// skip lines that are just a closing brace
	if bytes.Equal(trimmed, []byte("}")) {
		return "", "", false
	}
	// skip block openers like "name {"
	if bytes.HasSuffix(trimmed, []byte("{")) {
		return "", "", false
	}

	idx := bytes.Index(trimmed, []byte(" = "))
	if idx < 0 {
		// also try "=" without spaces
		idx = bytes.IndexByte(trimmed, '=')
		if idx < 0 {
			return "", "", false
		}
		k := string(bytes.TrimSpace(trimmed[:idx]))
		v := string(bytes.TrimSpace(trimmed[idx+1:]))
		return k, v, true
	}
	k := string(trimmed[:idx])
	v := string(trimmed[idx+3:])
	return k, v, true
}

// isBlockOpen checks if a line opens a named block: "name {" or "name{"
func isBlockOpen(trimmed []byte, name string) bool {
	s := string(trimmed)
	// match "name {" or "name{"
	if s == name+" {" || s == name+"{" {
		return true
	}
	return false
}

// braceBalance returns the net brace depth change for a line.
func braceBalance(trimmed []byte) int {
	bal := 0
	for _, b := range trimmed {
		switch b {
		case '{':
			bal++
		case '}':
			bal--
		}
	}
	return bal
}

func leadingWhitespace(line []byte) string {
	for i, b := range line {
		if b != ' ' && b != '\t' {
			return string(line[:i])
		}
	}
	return string(line)
}

func insertLine(lines [][]byte, at int, newLine []byte) [][]byte {
	lines = append(lines, nil)
	copy(lines[at+1:], lines[at:])
	lines[at] = newLine
	return lines
}

func removeLine(lines [][]byte, at int) [][]byte {
	return append(lines[:at], lines[at+1:]...)
}

func ensureTrailingNewline(data []byte) []byte {
	if len(data) == 0 || data[len(data)-1] != '\n' {
		return append(data, '\n')
	}
	return data
}
