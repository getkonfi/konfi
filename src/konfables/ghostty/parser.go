package ghostty

import "github.com/emin/konfigurator/pkg/parser"

func newParser() *parser.FlatParser {
	return &parser.FlatParser{Split: parser.SplitEquals, Format: parser.FormatEquals}
}
