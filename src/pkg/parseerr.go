package pkg

import (
	"errors"
	"fmt"
)

// ErrNestedNotSupported is returned when a flat parser is asked to handle nested paths.
var ErrNestedNotSupported = errors.New("nested path access not supported by this parser")

// ParseError describes a problem encountered while parsing config data.
type ParseError struct {
	Line    int
	Column  int
	Message string
	Source  string // file path or format name
}

func (e *ParseError) Error() string {
	loc := e.Source
	if e.Line > 0 {
		loc = fmt.Sprintf("%s:%d", loc, e.Line)
		if e.Column > 0 {
			loc = fmt.Sprintf("%s:%d", loc, e.Column)
		}
	}
	return fmt.Sprintf("%s: %s", loc, e.Message)
}

// IsParseError checks whether err (or a wrapped error) is a *ParseError.
func IsParseError(err error) bool {
	var pe *ParseError
	return errors.As(err, &pe)
}
