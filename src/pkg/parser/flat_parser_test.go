package parser

import (
	"testing"
)

func TestFlatParserFindValue(t *testing.T) {
	p := &FlatParser{Split: SplitEquals, Format: FormatEquals}
	data := []byte("font-size = 14\nbackground = 282828\n# comment\nforeground = ebdbb2\n")

	val, ok := p.FindValue(data, "font-size")
	if !ok || val != "14" {
		t.Errorf("FindValue(font-size) = %q, %v; want 14, true", val, ok)
	}
	_, ok = p.FindValue(data, "missing")
	if ok {
		t.Error("FindValue(missing) should return false")
	}
}

func TestFlatParserFindLine(t *testing.T) {
	p := &FlatParser{Split: SplitEquals, Format: FormatEquals}
	data := []byte("a = 1\nb = 2\nc = 3\n")

	line, ok := p.FindLine(data, "b")
	if !ok || line != 1 {
		t.Errorf("FindLine(b) = %d, %v; want 1, true", line, ok)
	}
	_, ok = p.FindLine(data, "missing")
	if ok {
		t.Error("FindLine(missing) should return false")
	}
}

func TestFlatParserSetValue(t *testing.T) {
	p := &FlatParser{Split: SplitEquals, Format: FormatEquals}

	// replace existing
	data := []byte("a = 1\nb = 2\n")
	out, err := p.SetValue(data, "a", "10")
	if err != nil {
		t.Fatal(err)
	}
	val, ok := p.FindValue(out, "a")
	if !ok || val != "10" {
		t.Errorf("after SetValue existing: got %q, %v", val, ok)
	}

	// append new
	out, err = p.SetValue(data, "c", "3")
	if err != nil {
		t.Fatal(err)
	}
	val, ok = p.FindValue(out, "c")
	if !ok || val != "3" {
		t.Errorf("after SetValue new: got %q, %v", val, ok)
	}
}

func TestFlatParserSetValueEmptyOldValue(t *testing.T) {
	p := &FlatParser{Split: SplitEquals, Format: FormatEquals}
	data := []byte("key =\n")
	out, err := p.SetValue(data, "key", "value")
	if err != nil {
		t.Fatal(err)
	}
	val, ok := p.FindValue(out, "key")
	if !ok || val != "value" {
		t.Errorf("got %q, %v; want value, true", val, ok)
	}
}

func TestFlatParserDeleteKey(t *testing.T) {
	p := &FlatParser{Split: SplitEquals, Format: FormatEquals}
	data := []byte("a = 1\nb = 2\nc = 3\n")

	out, err := p.DeleteKey(data, "b")
	if err != nil {
		t.Fatal(err)
	}
	_, ok := p.FindValue(out, "b")
	if ok {
		t.Error("b should be deleted")
	}
	val, ok := p.FindValue(out, "a")
	if !ok || val != "1" {
		t.Errorf("a should survive: got %q, %v", val, ok)
	}
}

func TestFlatParserFindAll(t *testing.T) {
	p := &FlatParser{Split: SplitEquals, Format: FormatEquals}
	data := []byte("a = 1\n# skip\nb = 2\n\n")

	m := p.FindAll(data)
	if len(m) != 2 {
		t.Fatalf("FindAll: got %d entries, want 2", len(m))
	}
	if m["a"] != "1" || m["b"] != "2" {
		t.Errorf("FindAll = %v", m)
	}
}

func TestFlatParserFindAllMulti(t *testing.T) {
	p := &FlatParser{Split: SplitEquals, Format: FormatEquals}
	data := []byte("keybind = ctrl+c=copy\nkeybind = ctrl+v=paste\nsingle = val\n")

	singles, multi := p.FindAllMulti(data)
	if len(singles) != 1 || singles["single"] != "val" {
		t.Errorf("singles = %v, want single=val", singles)
	}
	if len(multi) != 1 || len(multi["keybind"]) != 2 {
		t.Errorf("multi = %v, want keybind with 2 values", multi)
	}
	if multi["keybind"][0] != "ctrl+c=copy" || multi["keybind"][1] != "ctrl+v=paste" {
		t.Errorf("multi[keybind] = %v", multi["keybind"])
	}
}

func TestFlatParserListKeys(t *testing.T) {
	p := &FlatParser{Split: SplitEquals, Format: FormatEquals}
	data := []byte("a = 1\n# skip\nb = 2\n")

	keys := p.ListKeys(data)
	if len(keys) != 2 || keys[0] != "a" || keys[1] != "b" {
		t.Errorf("ListKeys = %v, want [a b]", keys)
	}
}

func TestFlatParserFindValues(t *testing.T) {
	p := &FlatParser{Split: SplitEquals, Format: FormatEquals}
	data := []byte("k = 1\nk = 2\nk = 3\nother = x\n")

	vals, ok := p.FindValues(data, "k")
	if !ok || len(vals) != 3 {
		t.Errorf("FindValues(k) = %v, %v; want 3 values", vals, ok)
	}
	_, ok = p.FindValues(data, "missing")
	if ok {
		t.Error("FindValues(missing) should return false")
	}
}

func TestFlatParserSetValues(t *testing.T) {
	p := &FlatParser{Split: SplitEquals, Format: FormatEquals}
	data := []byte("k = old\nother = x\n")

	out, err := p.SetValues(data, "k", []string{"new1", "new2"})
	if err != nil {
		t.Fatal(err)
	}
	vals, ok := p.FindValues(out, "k")
	if !ok || len(vals) != 2 {
		t.Errorf("SetValues: FindValues = %v, %v", vals, ok)
	}
	if vals[0] != "new1" || vals[1] != "new2" {
		t.Errorf("SetValues: values = %v", vals)
	}
	// other key should survive
	val, ok := p.FindValue(out, "other")
	if !ok || val != "x" {
		t.Errorf("other key should survive: got %q, %v", val, ok)
	}
}

func TestFlatParserSetValuesEmpty(t *testing.T) {
	p := &FlatParser{Split: SplitEquals, Format: FormatEquals}
	data := []byte("k = 1\nk = 2\nother = x\n")

	out, err := p.SetValues(data, "k", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := p.FindValues(out, "k")
	if ok {
		t.Error("SetValues with nil should remove all instances")
	}
}

func TestFlatParserEmptyData(t *testing.T) {
	p := &FlatParser{Split: SplitEquals, Format: FormatEquals}
	data := []byte("")

	_, ok := p.FindValue(data, "key")
	if ok {
		t.Error("FindValue on empty data should return false")
	}
	keys := p.ListKeys(data)
	if len(keys) != 0 {
		t.Errorf("ListKeys on empty = %v", keys)
	}
	out, err := p.SetValue(data, "new", "val")
	if err != nil {
		t.Fatal(err)
	}
	val, ok := p.FindValue(out, "new")
	if !ok || val != "val" {
		t.Errorf("SetValue on empty: got %q, %v", val, ok)
	}
}

func TestSplitEqualsOrSpace(t *testing.T) {
	tests := []struct {
		input string
		key   string
		val   string
		ok    bool
	}{
		{"key value", "key", "value", true},
		{"key=value", "key", "value", true},
		{"key ", "key", "", true},
		{"key", "key", "", true},
		{"", "", "", false},
		{"key with spaces", "key", "with spaces", true},
	}
	for _, tt := range tests {
		k, v, ok := SplitEqualsOrSpace(tt.input)
		if ok != tt.ok || k != tt.key || v != tt.val {
			t.Errorf("SplitEqualsOrSpace(%q) = %q, %q, %v; want %q, %q, %v",
				tt.input, k, v, ok, tt.key, tt.val, tt.ok)
		}
	}
}

func TestSplitColon(t *testing.T) {
	k, v, ok := SplitColon("key: value")
	if !ok || k != "key" || v != "value" {
		t.Errorf("SplitColon = %q, %q, %v", k, v, ok)
	}
	_, _, ok = SplitColon("no colon")
	if ok {
		t.Error("SplitColon should return false without colon")
	}
}

func TestSplitSpacedEquals(t *testing.T) {
	k, v, ok := SplitSpacedEquals("key = value")
	if !ok || k != "key" || v != "value" {
		t.Errorf("SplitSpacedEquals = %q, %q, %v", k, v, ok)
	}
	_, _, ok = SplitSpacedEquals("key=value")
	if ok {
		t.Error("SplitSpacedEquals should require spaces around =")
	}
}

func TestFormatFunctions(t *testing.T) {
	if got := FormatEquals("a", "b"); got != "a = b" {
		t.Errorf("FormatEquals = %q", got)
	}
	if got := FormatSpace("a", "b"); got != "a b" {
		t.Errorf("FormatSpace = %q", got)
	}
	if got := FormatColon("a", "b"); got != "a: b" {
		t.Errorf("FormatColon = %q", got)
	}
}

func TestFlatParserRoundTrip(t *testing.T) {
	p := &FlatParser{Split: SplitEquals, Format: FormatEquals}
	data := []byte("# config\na = 1\nb = 2\n\n")

	// set existing
	out, _ := p.SetValue(data, "a", "10")
	val, ok := p.FindValue(out, "a")
	if !ok || val != "10" {
		t.Fatalf("round-trip set: got %q, %v", val, ok)
	}
	// add new
	out, _ = p.SetValue(out, "c", "3")
	val, ok = p.FindValue(out, "c")
	if !ok || val != "3" {
		t.Fatalf("round-trip add: got %q, %v", val, ok)
	}
	// delete
	out, _ = p.DeleteKey(out, "b")
	_, ok = p.FindValue(out, "b")
	if ok {
		t.Fatal("round-trip delete: b should be gone")
	}
	// original untouched keys survive
	val, ok = p.FindValue(out, "a")
	if !ok || val != "10" {
		t.Fatalf("round-trip survival: got %q, %v", val, ok)
	}
	// comments survive
	if len(out) > 0 && string(out[:1]) != "#" {
		t.Errorf("comment should survive round-trip, got:\n%s", out)
	}
}
