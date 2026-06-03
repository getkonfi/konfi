package gnome

import "github.com/eminert/konfi/pkg/parser"

func newParser() *parser.FlatParser {
	return &parser.FlatParser{Split: parser.SplitSpacedEquals, Format: parser.FormatEquals}
}
