package starship

import "github.com/eminert/konfi/pkg/parser"

func newParser() *parser.SectionParser {
	return &parser.SectionParser{SplitKey: parser.SplitKeyFirst}
}
