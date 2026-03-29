package gnome

import "github.com/emin/konfigurator/pkg/parser"

func newParser() *parser.FlatParser {
	return &parser.FlatParser{Split: parser.SplitSpacedEquals, Format: parser.FormatEquals}
}
