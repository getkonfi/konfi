package dconf

import (
	"strings"

	"github.com/emin/konfigurator/pkg"
)

func newParser() *pkg.FlatParser {
	return &pkg.FlatParser{Split: pkg.SplitSpacedEquals, Format: pkg.FormatEquals}
}

// cutKV splits "key = value" on " = ". used by the persister's parseFlat.
func cutKV(s string) (key, value string, ok bool) {
	k, v, found := strings.Cut(s, " = ")
	if !found {
		return "", "", false
	}
	return k, v, true
}
