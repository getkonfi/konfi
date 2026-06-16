package gnome

import "github.com/getkonfi/konfi/pkg/parser"

func newParser() *parser.FlatParser {
	return &parser.FlatParser{Split: parser.SplitSpacedEquals, Format: parser.FormatEquals}
}
