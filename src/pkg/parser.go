package pkg

// ParserCapabilities describes what a parser format supports.
type ParserCapabilities struct {
	SupportsComments   bool
	SupportsNesting    bool
	LosslessRoundtrip  bool
	SupportsMultivalue bool
}
