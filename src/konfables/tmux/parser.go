package tmux

import (
	"strings"

	"github.com/eminert/konfi/pkg/parser"
)

// newParser handles the tmux config format: `set -g key value`,
// `set-option -g key value`, `setw -g key value`. it reuses FlatParser with a
// tmux-aware splitter; new keys are written as global `set -g key value` lines.
func newParser() *parser.FlatParser {
	return &parser.FlatParser{Split: parseTmuxSet, Format: formatTmuxSet}
}

// formatTmuxSet renders a new key as a global set-option line.
func formatTmuxSet(key, value string) string {
	return "set -g " + key + " " + value
}

// parseTmuxSet extracts key and value from a tmux set line.
// handles: set -g key value, set-option -g key value, setw -g key value
func parseTmuxSet(line string) (key, value string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed[0] == '#' {
		return "", "", false
	}

	fields := strings.Fields(trimmed)
	if len(fields) < 3 {
		return "", "", false
	}

	cmd := fields[0]
	if cmd != "set" && cmd != "set-option" && cmd != "set-window-option" && cmd != "setw" {
		return "", "", false
	}

	// skip flags (tokens starting with -)
	i := 1
	for i < len(fields) && strings.HasPrefix(fields[i], "-") {
		i++
	}

	if i >= len(fields) {
		return "", "", false
	}

	key = fields[i]
	i++

	if i < len(fields) {
		value = strings.Join(fields[i:], " ")
	}

	return key, value, true
}
