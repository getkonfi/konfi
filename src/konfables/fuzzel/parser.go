package fuzzel

import (
	"strings"

	cfgparse "github.com/eminert/konfi/pkg/parser"
)

type parser struct {
	base cfgparse.SectionParser
}

func newParser() *parser {
	return &parser{base: cfgparse.SectionParser{SplitKey: cfgparse.SplitKeyFirst, CommentChars: "#;"}}
}

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	if isMainKey(key) {
		if v, ok := p.base.FindValue(data, key); ok {
			return v, true
		}
		return p.base.FindValue(data, "main."+key)
	}
	return p.base.FindValue(data, key)
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	if isMainKey(key) {
		if line, ok := p.base.FindLine(data, key); ok {
			return line, true
		}
		return p.base.FindLine(data, "main."+key)
	}
	return p.base.FindLine(data, key)
}

func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	value = formatPromptValue(key, value)
	if isMainKey(key) {
		if _, _, found := cfgparse.FindTopLevelKey(data, key); found {
			return p.base.SetValue(data, key, value)
		}
		if hasSection(data, "main") {
			return p.base.SetValue(data, "main."+key, value)
		}
	}
	return p.base.SetValue(data, key, value)
}

func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	if isMainKey(key) {
		if _, _, found := cfgparse.FindTopLevelKey(data, key); found {
			return p.base.DeleteKey(data, key)
		}
		if _, _, found := cfgparse.FindKeyInSection(data, "main", key); found {
			return p.base.DeleteKey(data, "main."+key)
		}
	}
	return p.base.DeleteKey(data, key)
}

func (p *parser) ListKeys(data []byte) []string {
	keys := p.base.ListKeys(data)
	for i, key := range keys {
		keys[i] = normalizeMainKey(key)
	}
	return keys
}

func (p *parser) FindAll(data []byte) map[string]string {
	raw := p.base.FindAll(data)
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		out[normalizeMainKey(key)] = value
	}
	return out
}

func isMainKey(key string) bool {
	return !strings.Contains(key, ".")
}

func normalizeMainKey(key string) string {
	return strings.TrimPrefix(key, "main.")
}

func formatPromptValue(key, value string) string {
	if normalizeMainKey(key) != "prompt" || value == "" {
		return value
	}
	if strings.Trim(value, " \t") == value || isDoubleQuoted(value) {
		return value
	}
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func isDoubleQuoted(value string) bool {
	return len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"'
}

func hasSection(data []byte, section string) bool {
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed[0] != '[' {
			continue
		}
		if cfgparse.ParseSectionHeader(trimmed) == section {
			return true
		}
	}
	return false
}
