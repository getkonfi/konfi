package git

import "github.com/emin/konfigurator/pkg/parser"

func newParser() *parser.SectionParser {
	return &parser.SectionParser{SplitKey: parser.SplitKeyFirst, CommentChars: "#;"}
}
