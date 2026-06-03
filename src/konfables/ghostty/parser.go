package ghostty

import "github.com/eminert/konfi/pkg/parser"

func newParser() *parser.FlatParser {
	return &parser.FlatParser{Split: parser.SplitEquals, Format: parser.FormatEquals}
}
