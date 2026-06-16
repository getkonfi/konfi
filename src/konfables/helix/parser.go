package helix

import "github.com/getkonfi/konfi/pkg/parser"

func newParser() *parser.SectionParser {
	return &parser.SectionParser{SplitKey: parser.SplitKeyLast}
}
