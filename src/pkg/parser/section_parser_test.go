package parser

import (
	"bytes"
	"testing"
)

func TestSectionParserFindValue(t *testing.T) {
	p := &SectionParser{SplitKey: SplitKeyLast}
	data := []byte("toplevel = yes\n\n[colors]\nbackground = 282828\nforeground = ebdbb2\n")

	val, ok := p.FindValue(data, "toplevel")
	if !ok || val != "yes" {
		t.Errorf("FindValue(toplevel) = %q, %v", val, ok)
	}
	val, ok = p.FindValue(data, "colors.background")
	if !ok || val != "282828" {
		t.Errorf("FindValue(colors.background) = %q, %v", val, ok)
	}
	_, ok = p.FindValue(data, "missing.key")
	if ok {
		t.Error("FindValue(missing.key) should return false")
	}
}

func TestSectionParserFindLine(t *testing.T) {
	p := &SectionParser{SplitKey: SplitKeyLast}
	data := []byte("top = 1\n[sec]\nkey = val\n")

	line, ok := p.FindLine(data, "top")
	if !ok || line != 0 {
		t.Errorf("FindLine(top) = %d, %v; want 0, true", line, ok)
	}
	line, ok = p.FindLine(data, "sec.key")
	if !ok || line != 2 {
		t.Errorf("FindLine(sec.key) = %d, %v; want 2, true", line, ok)
	}
}

func TestSectionParserSetValue(t *testing.T) {
	p := &SectionParser{SplitKey: SplitKeyLast}

	// replace top-level
	data := []byte("top = old\n[sec]\nkey = old\n")
	out, err := p.SetValue(data, "top", "new")
	if err != nil {
		t.Fatal(err)
	}
	val, ok := p.FindValue(out, "top")
	if !ok || val != "new" {
		t.Errorf("SetValue top-level: got %q, %v", val, ok)
	}

	// replace in section
	out, err = p.SetValue(data, "sec.key", "new")
	if err != nil {
		t.Fatal(err)
	}
	val, ok = p.FindValue(out, "sec.key")
	if !ok || val != "new" {
		t.Errorf("SetValue in section: got %q, %v", val, ok)
	}

	// add new key to existing section
	out, err = p.SetValue(data, "sec.newkey", "val")
	if err != nil {
		t.Fatal(err)
	}
	val, ok = p.FindValue(out, "sec.newkey")
	if !ok || val != "val" {
		t.Errorf("SetValue new in section: got %q, %v", val, ok)
	}

	// add new section
	out, err = p.SetValue(data, "newsec.key", "val")
	if err != nil {
		t.Fatal(err)
	}
	val, ok = p.FindValue(out, "newsec.key")
	if !ok || val != "val" {
		t.Errorf("SetValue new section: got %q, %v", val, ok)
	}
}

func TestSectionParserDeleteKey(t *testing.T) {
	p := &SectionParser{SplitKey: SplitKeyLast}
	data := []byte("top = 1\n[sec]\nkey = val\nother = x\n")

	out, err := p.DeleteKey(data, "sec.key")
	if err != nil {
		t.Fatal(err)
	}
	_, ok := p.FindValue(out, "sec.key")
	if ok {
		t.Error("sec.key should be deleted")
	}
	// other survives
	val, ok := p.FindValue(out, "sec.other")
	if !ok || val != "x" {
		t.Errorf("sec.other should survive: got %q, %v", val, ok)
	}

	// delete missing is no-op
	out2, err := p.DeleteKey(data, "missing.key")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out2, data) {
		t.Error("deleting missing key should be no-op")
	}
}

func TestSectionParserFindAll(t *testing.T) {
	p := &SectionParser{SplitKey: SplitKeyLast}
	data := []byte("top = 1\n# skip\n[sec]\nkey = val\n")

	m := p.FindAll(data)
	if len(m) != 2 {
		t.Fatalf("FindAll: got %d entries, want 2", len(m))
	}
	if m["top"] != "1" || m["sec.key"] != "val" {
		t.Errorf("FindAll = %v", m)
	}
}

func TestSectionParserListKeys(t *testing.T) {
	p := &SectionParser{SplitKey: SplitKeyLast}
	data := []byte("top = 1\n[sec]\nkey = val\nother = x\n")

	keys := p.ListKeys(data)
	if len(keys) != 3 {
		t.Fatalf("ListKeys: got %d keys, want 3", len(keys))
	}
	expected := map[string]bool{"top": true, "sec.key": true, "sec.other": true}
	for _, k := range keys {
		if !expected[k] {
			t.Errorf("unexpected key: %q", k)
		}
	}
}

func TestSectionParserCommentChars(t *testing.T) {
	p := &SectionParser{SplitKey: SplitKeyFirst, CommentChars: "#;"}
	data := []byte("[core]\n;comment\nkey = val\n#also comment\n")

	keys := p.ListKeys(data)
	if len(keys) != 1 || keys[0] != "core.key" {
		t.Errorf("ListKeys with #; comments = %v", keys)
	}
}

func TestSectionParserEmptyData(t *testing.T) {
	p := &SectionParser{SplitKey: SplitKeyLast}
	data := []byte("")

	_, ok := p.FindValue(data, "key")
	if ok {
		t.Error("FindValue on empty should return false")
	}
	keys := p.ListKeys(data)
	if len(keys) != 0 {
		t.Errorf("ListKeys on empty = %v", keys)
	}
}

func TestSplitKeyLast(t *testing.T) {
	sec, field := SplitKeyLast("a.b.c")
	if sec != "a.b" || field != "c" {
		t.Errorf("SplitKeyLast(a.b.c) = %q, %q", sec, field)
	}
	sec, field = SplitKeyLast("no-dot")
	if sec != "" || field != "no-dot" {
		t.Errorf("SplitKeyLast(no-dot) = %q, %q", sec, field)
	}
}

func TestSplitKeyFirst(t *testing.T) {
	sec, field := SplitKeyFirst("a.b.c")
	if sec != "a" || field != "b.c" {
		t.Errorf("SplitKeyFirst(a.b.c) = %q, %q", sec, field)
	}
	sec, field = SplitKeyFirst("no-dot")
	if sec != "" || field != "no-dot" {
		t.Errorf("SplitKeyFirst(no-dot) = %q, %q", sec, field)
	}
}

func TestSectionParserRoundTrip(t *testing.T) {
	p := &SectionParser{SplitKey: SplitKeyLast}
	data := []byte("# header\n\ntop = yes\n\n[colors]\nbackground = 282828\nforeground = ebdbb2\n")

	// set existing
	out, _ := p.SetValue(data, "colors.background", "000000")
	val, ok := p.FindValue(out, "colors.background")
	if !ok || val != "000000" {
		t.Fatalf("round-trip set: got %q, %v", val, ok)
	}
	// add new to section
	out, _ = p.SetValue(out, "colors.cursor", "cccccc")
	val, ok = p.FindValue(out, "colors.cursor")
	if !ok || val != "cccccc" {
		t.Fatalf("round-trip add: got %q, %v", val, ok)
	}
	// delete
	out, _ = p.DeleteKey(out, "colors.foreground")
	_, ok = p.FindValue(out, "colors.foreground")
	if ok {
		t.Fatal("round-trip delete: foreground should be gone")
	}
	// untouched survive
	val, ok = p.FindValue(out, "top")
	if !ok || val != "yes" {
		t.Fatalf("round-trip survival: got %q, %v", val, ok)
	}
}
