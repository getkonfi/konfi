package konfables

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/getkonfi/konfi/pkg"
)

// FormatValue serializes a value for writing back to a config file. TOML
// string/color/enum/multi values need quoting and lists need array syntax;
// other formats write raw.
func FormatValue(value, fieldType, format string) string {
	if format == "toml" {
		switch fieldType {
		case "string", "color", "enum", "multi":
			return strconv.Quote(value)
		case "list":
			return formatTOMLList(value)
		}
	}
	if format == "zsh" {
		switch fieldType {
		case "string", "color", "enum", "multi":
			return quoteZsh(value)
		}
	}
	return value
}

var tomlNumberRE = regexp.MustCompile(`^[+-]?(?:0|[1-9](?:_?[0-9])*)(?:\.(?:[0-9](?:_?[0-9])*))?(?:[eE][+-]?(?:[0-9](?:_?[0-9])*))?$`)

func formatTOMLList(value string) string {
	items := SplitListValue(value)
	if len(items) == 0 {
		return "[]"
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, formatTOMLListItem(item))
	}
	return "[" + strings.Join(out, ", ") + "]"
}

func formatTOMLListItem(item string) string {
	item = strings.TrimSpace(item)
	if isRawTOMLValue(item) {
		return item
	}
	return strconv.Quote(item)
}

func isRawTOMLValue(value string) bool {
	if value == "" {
		return false
	}
	if len(value) >= 2 {
		switch {
		case value[0] == '"' && value[len(value)-1] == '"':
			return true
		case value[0] == '\'' && value[len(value)-1] == '\'':
			return true
		case value[0] == '[' && value[len(value)-1] == ']':
			return true
		case value[0] == '{' && value[len(value)-1] == '}':
			return true
		}
	}
	switch value {
	case "true", "false", "inf", "+inf", "-inf", "nan", "+nan", "-nan":
		return true
	}
	return tomlNumberRE.MatchString(value)
}

func quoteZsh(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

// SplitListValue parses a list-field value into items, accepting either the
// canonical "\n"-joined form (produced by the list editors) or the display
// ", "-joined form. items are trimmed and empties dropped.
func SplitListValue(s string) []string {
	if s == "" {
		return nil
	}
	var raw []string
	if strings.Contains(s, "\n") {
		raw = strings.Split(s, "\n")
	} else {
		raw = strings.Split(s, ", ")
	}
	out := raw[:0]
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

// isRawValueWidget reports whether a widget stores a raw value that must be
// written verbatim (no quoting, no list-splitting).
func isRawValueWidget(widget string) bool {
	switch widget {
	case "hook", "togglemap", "structlist", "blocklist", "rawtoml":
		return true
	}
	return false
}

// WriteField serializes value and writes it to f.Key in data, choosing the
// parser method that matches the field's shape: repeated-key lists via
// SetValues, raw-value widgets (hook/togglemap/structlist) via plain SetValue,
// and everything else via SetValue after format-specific quoting. it is the one
// place that maps a field's shape to a write strategy.
func WriteField(p Parser, data []byte, f pkg.Field, value, format string) ([]byte, error) {
	if isRawValueWidget(f.Widget) {
		return p.SetValue(data, f.Key, value)
	}
	if f.Type == "list" {
		if mvp, ok := p.(MultiValueParser); ok {
			return mvp.SetValues(data, f.Key, SplitListValue(value))
		}
	}
	return p.SetValue(data, f.Key, FormatValue(value, f.Type, format))
}
