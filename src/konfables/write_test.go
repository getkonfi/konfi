package konfables

import (
	"reflect"
	"testing"

	"github.com/eminert/konfi/pkg"
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

func TestFormatValueWritesTOMLListArray(t *testing.T) {
	got := FormatValue("--login\n-c\n80\ntrue\n\"already quoted\"", "list", "toml")
	want := `["--login", "-c", 80, true, "already quoted"]`
	if got != want {
		t.Errorf("FormatValue toml list = %q, want %q", got, want)
	}
	if got := FormatValue("", "list", "toml"); got != "[]" {
		t.Errorf("FormatValue empty toml list = %q, want []", got)
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

func TestWriteFieldRawWidgetUsesSetValueBeforeSetValues(t *testing.T) {
	p := &recordingParser{}
	field := pkg.Field{Key: "mas", Type: "list", Widget: "structlist"}
	value := "Xcode | 497799835\nThings | 904280696"

	if _, err := WriteField(p, nil, field, value, "brew"); err != nil {
		t.Fatal(err)
	}
	if p.used != "set" {
		t.Fatalf("WriteField used %q, want SetValue", p.used)
	}
	if p.setKey != "mas" || p.setValue != value {
		t.Fatalf("SetValue got key=%q value=%q", p.setKey, p.setValue)
	}
}

func TestWriteFieldPlainListUsesSetValues(t *testing.T) {
	p := &recordingParser{}
	field := pkg.Field{Key: "brew", Type: "list"}

	if _, err := WriteField(p, nil, field, "git\nripgrep", "brew"); err != nil {
		t.Fatal(err)
	}
	if p.used != "setValues" {
		t.Fatalf("WriteField used %q, want SetValues", p.used)
	}
	if p.setValuesKey != "brew" || !reflect.DeepEqual(p.setValues, []string{"git", "ripgrep"}) {
		t.Fatalf("SetValues got key=%q values=%v", p.setValuesKey, p.setValues)
	}
}

func TestWriteFieldRawTOMLWidgetBypassesQuoting(t *testing.T) {
	p := &recordingParser{}
	field := pkg.Field{Key: "fonts.extras", Type: "string", Widget: "rawtoml"}
	value := `[{ family = "Noto Color Emoji" }]`

	if _, err := WriteField(p, nil, field, value, "toml"); err != nil {
		t.Fatal(err)
	}
	if p.used != "set" {
		t.Fatalf("WriteField used %q, want SetValue", p.used)
	}
	if p.setValue != value {
		t.Fatalf("SetValue value = %q, want raw %q", p.setValue, value)
	}
}

type recordingParser struct {
	used         string
	setKey       string
	setValue     string
	setValuesKey string
	setValues    []string
}

func (p *recordingParser) FindValue([]byte, string) (string, bool) { return "", false }
func (p *recordingParser) FindLine([]byte, string) (int, bool)     { return -1, false }
func (p *recordingParser) DeleteKey(data []byte, _ string) ([]byte, error) {
	return data, nil
}
func (p *recordingParser) ListKeys([]byte) []string { return nil }
func (p *recordingParser) SetValue(data []byte, key, value string) ([]byte, error) {
	p.used = "set"
	p.setKey = key
	p.setValue = value
	return data, nil
}
func (p *recordingParser) FindValues([]byte, string) ([]string, bool) { return nil, false }
func (p *recordingParser) SetValues(data []byte, key string, values []string) ([]byte, error) {
	p.used = "setValues"
	p.setValuesKey = key
	p.setValues = values
	return data, nil
}
