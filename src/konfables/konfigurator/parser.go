package konfigurator

import "github.com/eminert/konfi/pkg/parser"

func newParser() *parser.FlatParser {
	return &parser.FlatParser{Split: parser.SplitColon, Format: parser.FormatColon}
}
