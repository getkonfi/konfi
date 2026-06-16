package starship

import "github.com/getkonfi/konfi/pkg/parser"

func newParser() *parser.SectionParser {
	return &parser.SectionParser{SplitKey: parser.SplitKeyFirst}
}
