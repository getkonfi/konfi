package brew

import (
	"fmt"
	"regexp"
	"strings"
)

// parser handles the Homebrew Bundle Brewfile (a line-oriented Ruby DSL).
// it models the common entry types and edits them surgically: lines it does
// not recognize (comments, blanks, ruby conditionals, cask_args, whalebrew,
// inline modifiers) are preserved verbatim.
//
//	tap "user/repo"
//	brew "name"[, ...modifiers]
//	cask "name"[, ...modifiers]
//	mas "Name", id: 12345
//	vscode "publisher.ext"
type parser struct{}

const masKey = "mas"

// listKeywords are entry types edited as plain string lists via MultiValueParser.
var listKeywords = map[string]bool{
	"tap":    true,
	"brew":   true,
	"cask":   true,
	"vscode": true,
}

// entryRe matches `keyword "value"<rest>`, capturing the leading keyword, the
// first quoted argument, and any trailing modifiers. single and double quotes
// are both accepted.
var entryRe = regexp.MustCompile(`^(\w+)\s+["']([^"']*)["'](.*)$`)

// idRe extracts the numeric App Store id from a mas line's trailing modifiers.
var idRe = regexp.MustCompile(`id:\s*(\d+)`)

// parseEntry classifies a single line. ok is false for comments, blanks, and
// any line that is not a `keyword "value"` entry.
func parseEntry(line string) (keyword, value, rest string, ok bool) {
	t := strings.TrimSpace(line)
	if t == "" || strings.HasPrefix(t, "#") {
		return "", "", "", false
	}
	m := entryRe.FindStringSubmatch(t)
	if m == nil {
		return "", "", "", false
	}
	return m[1], m[2], m[3], true
}

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	if key == masKey {
		return findMasValue(data)
	}
	vals, ok := p.FindValues(data, key)
	if !ok {
		return "", false
	}
	return vals[0], true
}

func (p *parser) FindValues(data []byte, key string) ([]string, bool) {
	if !listKeywords[key] {
		return nil, false
	}
	var vals []string
	for line := range strings.SplitSeq(string(data), "\n") {
		kw, v, _, ok := parseEntry(line)
		if ok && kw == key {
			vals = append(vals, v)
		}
	}
	if len(vals) == 0 {
		return nil, false
	}
	return vals, true
}

func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	if key == masKey {
		return setMasValue(data, value), nil
	}
	if listKeywords[key] {
		return p.SetValues(data, key, splitLines(value))
	}
	return data, nil
}

func (p *parser) SetValues(data []byte, key string, values []string) ([]byte, error) {
	if !listKeywords[key] {
		return data, fmt.Errorf("brew: %q is not a list field", key)
	}
	desired := dedupe(values)
	render := func(v string) string { return key + " " + quote(v) }
	lines := reconcile(strings.Split(string(data), "\n"), key, desired, render, false)
	return []byte(strings.Join(lines, "\n")), nil
}

func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		kw, _, _, ok := parseEntry(line)
		if ok && kw == key {
			continue
		}
		out = append(out, line)
	}
	return []byte(strings.Join(out, "\n")), nil
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	for i, line := range strings.Split(string(data), "\n") {
		kw, _, _, ok := parseEntry(line)
		if ok && kw == key {
			return i, true
		}
	}
	return -1, false
}

func (p *parser) ListKeys(data []byte) []string {
	seen := make(map[string]bool)
	var keys []string
	for line := range strings.SplitSeq(string(data), "\n") {
		kw, _, _, ok := parseEntry(line)
		if !ok || seen[kw] {
			continue
		}
		seen[kw] = true
		keys = append(keys, kw)
	}
	return keys
}

// findMasValue serializes mas entries into the structlist form: one
// "name | id" row per line.
func findMasValue(data []byte) (string, bool) {
	var rows []string
	for line := range strings.SplitSeq(string(data), "\n") {
		kw, val, rest, ok := parseEntry(line)
		if !ok || kw != masKey {
			continue
		}
		rows = append(rows, val+" | "+extractID(rest))
	}
	if len(rows) == 0 {
		return "", false
	}
	return strings.Join(rows, "\n"), true
}

// setMasValue rewrites mas entries from the structlist form. unlike the list
// fields, mas rows are always re-rendered because the id is editable.
func setMasValue(data []byte, value string) []byte {
	names := make([]string, 0)
	idByName := make(map[string]string)
	for line := range strings.SplitSeq(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name, id, _ := strings.Cut(line, "|")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, dup := idByName[name]; !dup {
			names = append(names, name)
		}
		idByName[name] = strings.TrimSpace(id)
	}
	render := func(name string) string {
		if id := idByName[name]; id != "" {
			return "mas " + quote(name) + ", id: " + id
		}
		return "mas " + quote(name)
	}
	lines := reconcile(strings.Split(string(data), "\n"), masKey, names, render, true)
	return []byte(strings.Join(lines, "\n"))
}

func extractID(rest string) string {
	if m := idRe.FindStringSubmatch(rest); m != nil {
		return m[1]
	}
	return ""
}

// reconcile rewrites every line of keyword so the file contains exactly the
// desired entries (matched by their quoted value) in desired order. existing
// entries keep their original line verbatim unless rerender is set; entries
// not in desired are dropped; new entries are inserted after the last existing
// line of that keyword, or appended at end. all other lines are untouched.
func reconcile(lines []string, keyword string, desired []string, render func(key string) string, rerender bool) []string {
	want := make(map[string]bool, len(desired))
	for _, k := range desired {
		want[k] = true
	}

	out := make([]string, 0, len(lines)+len(desired))
	emitted := make(map[string]bool, len(desired))
	lastIdx := -1
	for _, line := range lines {
		kw, val, _, ok := parseEntry(line)
		if ok && kw == keyword {
			if want[val] && !emitted[val] {
				if rerender {
					out = append(out, render(val))
				} else {
					out = append(out, line)
				}
				emitted[val] = true
				lastIdx = len(out) - 1
			}
			continue
		}
		out = append(out, line)
	}

	var fresh []string
	for _, k := range desired {
		if !emitted[k] {
			fresh = append(fresh, render(k))
			emitted[k] = true
		}
	}
	if len(fresh) == 0 {
		return out
	}
	if lastIdx < 0 {
		return appendAtEnd(out, fresh)
	}
	res := make([]string, 0, len(out)+len(fresh))
	res = append(res, out[:lastIdx+1]...)
	res = append(res, fresh...)
	res = append(res, out[lastIdx+1:]...)
	return res
}

// appendAtEnd appends fresh lines, preserving a single trailing blank line so
// the file keeps its terminating newline.
func appendAtEnd(out, fresh []string) []string {
	trailing := false
	if len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
		trailing = true
	}
	out = append(out, fresh...)
	if trailing {
		out = append(out, "")
	}
	return out
}

func quote(s string) string { return "\"" + s + "\"" }

func dedupe(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func splitLines(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, "\n")
}
