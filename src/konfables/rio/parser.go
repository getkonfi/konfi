package rio

import "github.com/emin/konfigurator/pkg"

func newParser() *pkg.SectionParser {
	return &pkg.SectionParser{SplitKey: pkg.SplitKeyLast}
}
