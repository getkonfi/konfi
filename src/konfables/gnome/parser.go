package gnome

import "github.com/emin/konfigurator/pkg"

func newParser() *pkg.FlatParser {
	return &pkg.FlatParser{Split: pkg.SplitSpacedEquals, Format: pkg.FormatEquals}
}
