package editors

import (
	"testing"

	"charm.land/bubbles/v2/textinput"
)

// TestHookEditorFilterCorruptsOptions demonstrates that filterMatcherCompletions
// corrupts matcherOptions via shared backing array after an empty-query alias.
func TestHookEditorFilterCorruptsOptions(t *testing.T) {
	e := &hookEditor{}
	e.matcherOptions = []string{"Bash", "Python", "Node"}
	e.input = textinput.New()

	// step 1: empty query — matcherFiltered aliases matcherOptions
	e.input.SetValue("")
	e.filterMatcherCompletions()

	// step 2: query that matches only the last element — writes into shared backing array
	e.input.SetValue("no")
	e.filterMatcherCompletions()

	// step 3: clear query — matcherFiltered = matcherOptions again, exposing corruption
	e.input.SetValue("")
	e.filterMatcherCompletions()

	// matcherOptions should still be ["Bash", "Python", "Node"]
	if len(e.matcherOptions) != 3 {
		t.Fatalf("matcherOptions length changed: got %d, want 3", len(e.matcherOptions))
	}
	if e.matcherOptions[0] != "Bash" {
		t.Errorf("matcherOptions[0] corrupted: got %q, want %q", e.matcherOptions[0], "Bash")
	}
	if e.matcherOptions[1] != "Python" {
		t.Errorf("matcherOptions[1] corrupted: got %q, want %q", e.matcherOptions[1], "Python")
	}
	if e.matcherOptions[2] != "Node" {
		t.Errorf("matcherOptions[2] corrupted: got %q, want %q", e.matcherOptions[2], "Node")
	}

	// matcherFiltered should contain all 3 original options
	if len(e.matcherFiltered) != 3 {
		t.Errorf("matcherFiltered length: got %d, want 3", len(e.matcherFiltered))
	}
}
