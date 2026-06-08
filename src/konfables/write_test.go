package konfables

import (
	"reflect"
	"testing"
)

// regression: list-field undo used to corrupt repeated keys because the
// EditOp's OldValue was display-form (", "-joined) while the apply path
// split on "\n" — collapsing every item into a single comma-laden string.
func TestSplitListValueAcceptsBothSeparators(t *testing.T) {
	want := []string{"foo", "bar", "baz"}

	cases := map[string]string{
		"newline-joined":     "foo\nbar\nbaz",
		"comma-space-joined": "foo, bar, baz",
		"with-trim":          "  foo \n  bar\nbaz  ",
		"with-empties":       "foo\n\nbar\nbaz\n",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			got := SplitListValue(in)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("SplitListValue(%q) = %v, want %v", in, got, want)
			}
		})
	}

	if got := SplitListValue(""); got != nil {
		t.Errorf("SplitListValue(\"\") = %v, want nil", got)
	}

	// when the value carries the canonical "\n", we must NOT also try
	// to re-split on ", ", or commas inside an item would split it.
	in := "ctrl+a, copy\nctrl+v, paste"
	got := SplitListValue(in)
	wantPair := []string{"ctrl+a, copy", "ctrl+v, paste"}
	if !reflect.DeepEqual(got, wantPair) {
		t.Errorf("SplitListValue(%q) = %v, want %v", in, got, wantPair)
	}
}

func TestFormatValueQuotesTOMLStrings(t *testing.T) {
	if got := FormatValue("hi there", "string", "toml"); got != `"hi there"` {
		t.Errorf(`FormatValue toml string = %q, want %q`, got, `"hi there"`)
	}
	// non-toml formats write raw
	if got := FormatValue("hi there", "string", "equals"); got != "hi there" {
		t.Errorf("FormatValue equals string = %q, want raw", got)
	}
	// numbers/bools are never quoted, even in toml
	if got := FormatValue("14", "number", "toml"); got != "14" {
		t.Errorf("FormatValue toml number = %q, want raw", got)
	}
}

func TestFormatValueQuotesZshStrings(t *testing.T) {
	if got := FormatValue("${P9K_CONTENT}", "string", "zsh"); got != "'${P9K_CONTENT}'" {
		t.Errorf("FormatValue zsh string = %q", got)
	}
	if got := FormatValue("don't", "string", "zsh"); got != "'don'\\''t'" {
		t.Errorf("FormatValue zsh single quote = %q", got)
	}
	if got := FormatValue("true", "bool", "zsh"); got != "true" {
		t.Errorf("FormatValue zsh bool = %q, want raw", got)
	}
	if got := FormatValue("42", "number", "zsh"); got != "42" {
		t.Errorf("FormatValue zsh number = %q, want raw", got)
	}
}
