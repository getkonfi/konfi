package dconf

import (
	"strings"

	"github.com/emin/konfigurator/pkg/parser"
)

func newParser() *parser.FlatParser {
	return &parser.FlatParser{Split: parser.SplitSpacedEquals, Format: parser.FormatEquals}
}

// cutKV splits "key = value" on " = ". used by the persister's parseFlat.
func cutKV(s string) (key, value string, ok bool) {
	k, v, found := strings.Cut(s, " = ")
	if !found {
		return "", "", false
	}
	return k, v, true
}
