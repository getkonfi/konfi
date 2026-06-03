package helix

import "github.com/eminert/konfi/pkg/parser"

func newParser() *parser.SectionParser {
	return &parser.SectionParser{SplitKey: parser.SplitKeyLast}
}
