package konfi

import "github.com/getkonfi/konfi/pkg/parser"

func newParser() *parser.FlatParser {
	return &parser.FlatParser{Split: parser.SplitColon, Format: parser.FormatColon}
}
