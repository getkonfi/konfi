package git

import (
	"strings"

	cfgparse "github.com/eminert/konfi/pkg/parser"
)

type parser struct {
	base cfgparse.SectionParser
}

func newParser() *parser {
	return &parser{base: cfgparse.SectionParser{SplitKey: splitGitKey, CommentChars: "#;"}}
}

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	if val, ok := p.base.FindValue(data, key); ok {
		return val, true
	}
	if legacySection, legacyField, ok := legacyGitKey(key); ok {
		legacy := cfgparse.SectionParser{SplitKey: cfgparse.SplitKeyFirst, CommentChars: "#;"}
		return legacy.FindValue(data, legacySection+".diff."+legacyField)
	}
	return "", false
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	if line, ok := p.base.FindLine(data, key); ok {
		return line, true
	}
	if legacySection, legacyField, ok := legacyGitKey(key); ok {
		legacy := cfgparse.SectionParser{SplitKey: cfgparse.SplitKeyFirst, CommentChars: "#;"}
		return legacy.FindLine(data, legacySection+".diff."+legacyField)
	}
	return -1, false
}

func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	if legacySection, legacyField, ok := legacyGitKey(key); ok {
		_, _, found := cfgparse.FindKeyInSection(data, gitSubsection(legacySection, "diff"), legacyField)
		if !found {
			if _, lineIdx, legacyFound := cfgparse.FindKeyInSection(data, legacySection, "diff."+legacyField); legacyFound {
				data = cfgparse.DeleteKeyOnLine(data, lineIdx)
			}
		}
	}
	return p.base.SetValue(data, key, value)
}

func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	out, err := p.base.DeleteKey(data, key)
	if err != nil {
		return out, err
	}
	if legacySection, legacyField, ok := legacyGitKey(key); ok {
		legacy := cfgparse.SectionParser{SplitKey: cfgparse.SplitKeyFirst, CommentChars: "#;"}
		return legacy.DeleteKey(out, legacySection+".diff."+legacyField)
	}
	return out, nil
}

func (p *parser) FindAll(data []byte) map[string]string {
	lines := strings.Split(string(data), "\n")
	m := make(map[string]string)
	currentSection := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isGitComment(trimmed[0]) {
			continue
		}
		if trimmed[0] == '[' {
			currentSection = cfgparse.ParseSectionHeader(trimmed)
			continue
		}
		k, v, ok := cfgparse.ParseKVLine(trimmed)
		if !ok {
			continue
		}
		m[gitConfigKey(currentSection, k)] = v
	}
	return m
}

func (p *parser) ListKeys(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	var keys []string
	currentSection := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isGitComment(trimmed[0]) {
			continue
		}
		if trimmed[0] == '[' {
			currentSection = cfgparse.ParseSectionHeader(trimmed)
			continue
		}
		k, _, ok := cfgparse.ParseKVLine(trimmed)
		if !ok {
			continue
		}
		keys = append(keys, gitConfigKey(currentSection, k))
	}
	return keys
}

func splitGitKey(key string) (section, field string) {
	if section, field, ok := gitSubsectionKey(key); ok {
		return section, field
	}
	return cfgparse.SplitKeyFirst(key)
}

func gitSubsectionKey(key string) (section, field string, ok bool) {
	parts := strings.Split(key, ".")
	if len(parts) != 3 {
		return "", "", false
	}
	if parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", false
	}
	return gitSubsection(parts[0], parts[1]), parts[2], true
}

func gitSubsection(section, subsection string) string {
	return section + ` "` + subsection + `"`
}

func legacyGitKey(key string) (section, field string, ok bool) {
	section, field, ok = gitSubsectionKey(key)
	if !ok || !strings.EqualFold(section, gitSubsection("color", "diff")) {
		return "", "", false
	}
	return "color", field, true
}

func gitConfigKey(section, key string) string {
	if section == "" {
		return key
	}
	if base, subsection, ok := parseGitSubsection(section); ok {
		return base + "." + subsection + "." + key
	}
	return section + "." + key
}

func parseGitSubsection(section string) (base, subsection string, ok bool) {
	idx := strings.IndexAny(section, " \t")
	if idx < 0 {
		return "", "", false
	}
	base = strings.TrimSpace(section[:idx])
	subsection = strings.TrimSpace(section[idx+1:])
	if base == "" || subsection == "" {
		return "", "", false
	}
	if len(subsection) >= 2 && subsection[0] == '"' && subsection[len(subsection)-1] == '"' {
		subsection = subsection[1 : len(subsection)-1]
	}
	if subsection == "" {
		return "", "", false
	}
	return base, subsection, true
}

func isGitComment(c byte) bool {
	return c == '#' || c == ';'
}
