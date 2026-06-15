package hyprland

import "github.com/eminert/konfi/pkg/parser"

func newParser() *parser.HyprParser {
	return parser.NewHyprParser()
}
