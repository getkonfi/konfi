package gtk

import (
	"strings"

	cfgparse "github.com/getkonfi/konfi/pkg/parser"
)

// parser handles GTK settings.ini (GKeyFile format): a [Settings] section with
// `key=value` pairs. it wraps SectionParser for find/replace/delete but inserts
// new keys without spaces around `=` to match GKeyFile's canonical style.
type parser struct {
	base cfgparse.SectionParser
}

func newParser() *parser {
	return &parser{base: cfgparse.SectionParser{SplitKey: cfgparse.SplitKeyFirst}}
}

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	return p.base.FindValue(data, key)
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	return p.base.FindLine(data, key)
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

// SetValue replaces an existing value in place (preserving the line's spacing)
// or inserts a new `key=value` line without spaces around `=`.
func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	section, field := cfgparse.SplitKeyFirst(key)
	if _, lineIdx, found := cfgparse.FindKeyInSection(data, section, field); found {
		return cfgparse.ReplaceValueOnLine(data, lineIdx, value), nil
	}
	return insertKeyNoSpace(data, section, field, value), nil
}

// insertKeyNoSpace appends `field=value` to the end of [section], creating the
// section at the end of the file if absent. mirrors parser.InsertKeyInSection
// but emits `key=value` rather than `key = value`.
func insertKeyNoSpace(data []byte, section, field, value string) []byte {
	lines := strings.Split(string(data), "\n")
	line := field + "=" + value
	sectionEnd := -1
	inSection := false

	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" && trimmed[0] == '[' {
			if inSection {
				return insertLine(lines, i, line)
			}
			inSection = cfgparse.ParseSectionHeader(trimmed) == section
			if inSection {
				sectionEnd = i
			}
			continue
		}
		if inSection && trimmed != "" {
			sectionEnd = i
		}
	}

	if inSection && sectionEnd >= 0 {
		return insertLine(lines, sectionEnd+1, line)
	}

	result := string(data)
	if result != "" && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	result += "[" + section + "]\n" + line + "\n"
	return []byte(result)
}

func insertLine(lines []string, at int, content string) []byte {
	if at > len(lines) {
		at = len(lines)
	}
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:at]...)
	out = append(out, content)
	out = append(out, lines[at:]...)
	return []byte(strings.Join(out, "\n"))
}
