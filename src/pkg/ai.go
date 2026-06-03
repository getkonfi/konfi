package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// askDefaultTimeout caps any AskClaude call whose caller forgot to set a
// deadline on the context. without this, a hung claude subprocess would
// block the TUI's loading spinner indefinitely.
const askDefaultTimeout = 30 * time.Second

// AISuggestion represents a single AI-recommended config option.
type AISuggestion struct {
	App     string `json:"app"`
	Section string `json:"section"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	Reason  string `json:"reason"`
}

// SerializeSchemas builds a compact text representation of all schemas
// for use as AI context. pipe-delimited, one line per field.
func SerializeSchemas(schemas map[string]*Schema) string {
	var b strings.Builder
	b.WriteString("app|section|key|type|default|description|options\n")
	for _, schema := range schemas {
		app := schema.App
		for _, sec := range schema.Sections {
			for fi := range sec.Fields {
			f := &sec.Fields[fi]
				desc := strings.ReplaceAll(f.Description, "\n", " ")
				if len(desc) > 120 {
					desc = desc[:120]
				}
				opts := ""
				switch {
				case len(f.Options) > 0:
					opts = strings.Join(f.Options, ",")
				case f.Min != nil && f.Max != nil:
					opts = fmt.Sprintf("%g-%g", *f.Min, *f.Max)
				case f.Min != nil:
					opts = fmt.Sprintf("%g+", *f.Min)
				case f.Max != nil:
					opts = fmt.Sprintf("-%g", *f.Max)
				}
				fmt.Fprintf(&b, "%s|%s|%s|%s|%s|%s|%s\n",
					app, sec.Name, f.Key, f.Type, f.Default, desc, opts)
			}
		}
	}
	return b.String()
}

const askPrompt = `you are a dotfile configuration expert embedded in konfigurator, a TUI for editing dotfiles.
given the application schemas below, find configuration options matching the user's intent.
respond ONLY with a JSON array. no markdown fences, no explanation outside the array.

each element: {"app":"name","section":"section","key":"field_key","value":"suggested_value","reason":"brief explanation"}

rules:
- return 1-8 most relevant results, ranked by relevance
- include results from multiple apps when applicable
- suggest concrete values when the intent implies one
- if the field is an enum, the value must be from the options list
- if the field is a number with min/max, the value must be in range
- if you cannot suggest a value, set value to ""
- keep reasons under 80 characters

<schemas>
%s</schemas>

intent: %s`

// AskClaude runs claude -p with schema context and a user query.
// when ctx has no deadline, a 30s default is applied so the TUI can't hang.
func AskClaude(ctx context.Context, schemas map[string]*Schema, query string) ([]AISuggestion, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, askDefaultTimeout)
		defer cancel()
	}

	schemaText := SerializeSchemas(schemas)
	prompt := fmt.Sprintf(askPrompt, schemaText, query)

	cmd := exec.CommandContext(ctx, "claude", "-p")
	cmd.Stdin = strings.NewReader(prompt)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("claude: %w", err)
	}

	text := strings.TrimSpace(string(out))
	// strip markdown fences if claude adds them despite instructions
	if strings.HasPrefix(text, "```") {
		if i := strings.Index(text[3:], "\n"); i >= 0 {
			text = text[3+i+1:]
		}
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}

	var results []AISuggestion
	if err := json.Unmarshal([]byte(text), &results); err != nil {
		return nil, fmt.Errorf("parse response: %w\nraw: %s", err, truncateRaw(text, 200))
	}
	return results, nil
}

func truncateRaw(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
