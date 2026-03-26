package ghostty

import "github.com/emin/konfigurator/pkg"

func newParser() *pkg.FlatParser {
	return &pkg.FlatParser{Split: pkg.SplitEquals, Format: pkg.FormatEquals}
}
