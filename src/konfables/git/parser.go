package git

import (
	"strings"

	"github.com/emin/konfigurator/pkg"
)

// parser handles INI-style git config: [section] headers with key = value pairs.
// reuses the TOML line-level helpers since the format is structurally identical.
type parser struct{}

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	section, field := splitKey(key)
	if section == "" {
		val, _, found := pkg.FindTopLevelKey(data, field)
		return val, found
	}
	val, _, found := pkg.FindKeyInSection(data, section, field)
	return val, found
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	section, field := splitKey(key)
	if section == "" {
		_, lineIdx, found := pkg.FindTopLevelKey(data, field)
		return lineIdx, found
	}
	_, lineIdx, found := pkg.FindKeyInSection(data, section, field)
	return lineIdx, found
}

func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	section, field := splitKey(key)

	if section == "" {
		_, lineIdx, found := pkg.FindTopLevelKey(data, field)
		if found {
			return pkg.ReplaceValueOnLine(data, lineIdx, value), nil
		}
		return pkg.InsertTopLevelKey(data, field, value), nil
	}

	_, lineIdx, found := pkg.FindKeyInSection(data, section, field)
	if found {
		return pkg.ReplaceValueOnLine(data, lineIdx, value), nil
	}
	return pkg.InsertKeyInSection(data, section, field, value), nil
}

func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	section, field := splitKey(key)

	if section == "" {
		_, lineIdx, found := pkg.FindTopLevelKey(data, field)
		if !found {
			return data, nil
		}
		return pkg.DeleteKeyOnLine(data, lineIdx), nil
	}

	_, lineIdx, found := pkg.FindKeyInSection(data, section, field)
	if !found {
		return data, nil
	}
	return pkg.DeleteKeyOnLine(data, lineIdx), nil
}

func (p *parser) ListKeys(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	var keys []string
	currentSection := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed[0] == '#' || trimmed[0] == ';' {
			continue
		}
		if trimmed[0] == '[' {
			currentSection = pkg.ParseSectionHeader(trimmed)
			continue
		}
		k, _, ok := pkg.ParseKVLine(trimmed)
		if !ok {
			continue
		}
		if currentSection != "" {
			keys = append(keys, currentSection+"."+k)
		} else {
			keys = append(keys, k)
		}
	}
	return keys
}

func splitKey(key string) (section, field string) {
	idx := strings.IndexByte(key, '.')
	if idx < 0 {
		return "", key
	}
	return key[:idx], key[idx+1:]
}
