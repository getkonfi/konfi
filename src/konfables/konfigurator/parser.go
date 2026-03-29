package konfigurator

import "github.com/emin/konfigurator/pkg/parser"

func newParser() *parser.FlatParser {
	return &parser.FlatParser{Split: parser.SplitColon, Format: parser.FormatColon}
}
