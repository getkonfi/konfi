package kitty

import "github.com/eminert/konfi/pkg/parser"

func newParser() *parser.FlatParser {
	return &parser.FlatParser{Split: parser.SplitEqualsOrSpace, Format: parser.FormatSpace}
}
