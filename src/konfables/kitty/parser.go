package kitty

import "github.com/emin/konfigurator/pkg/parser"

func newParser() *parser.FlatParser {
	return &parser.FlatParser{Split: parser.SplitEqualsOrSpace, Format: parser.FormatSpace}
}
