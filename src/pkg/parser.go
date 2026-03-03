package pkg

// ParserCapabilities describes what a parser format supports.
type ParserCapabilities struct {
	SupportsComments   bool
	SupportsNesting    bool
	LosslessRoundtrip  bool
	SupportsMultivalue bool
}

// NestedParser is an optional interface for parsers that support
// hierarchical path-based access (e.g. YAML, JSONC, KDL, nested TOML).
// parsers for flat key=value formats do not need to implement this.
type NestedParser interface {
	FindValueAtPath(data []byte, path []string) (string, bool)
	SetValueAtPath(data []byte, path []string, value string) ([]byte, error)
}

// CapabilityReporter is an optional interface for parsers that advertise
// their format capabilities. parsers that don't implement this are assumed
// to have zero-value (all false) capabilities.
type CapabilityReporter interface {
	Capabilities() ParserCapabilities
}

// BaseParser provides default implementations for NestedParser and
// CapabilityReporter. embed it in concrete parsers to satisfy the
// optional interfaces without writing boilerplate.
type BaseParser struct{}

// FindValueAtPath returns ("", false) — flat parsers don't support path access.
func (BaseParser) FindValueAtPath(_ []byte, _ []string) (string, bool) {
	return "", false
}

// SetValueAtPath returns ErrNestedNotSupported — flat parsers can't handle paths.
func (BaseParser) SetValueAtPath(data []byte, _ []string, _ string) ([]byte, error) {
	return data, ErrNestedNotSupported
}

// Capabilities returns zero-value capabilities (all false).
func (BaseParser) Capabilities() ParserCapabilities {
	return ParserCapabilities{}
}
