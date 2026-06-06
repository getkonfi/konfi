package parser

import "testing"

func TestMultilineRead(t *testing.T) {
	p := &SectionParser{SplitKey: SplitKeyFirst}
	cfg := []byte("add_newline = true\nformat = \"\"\"\n$directory\\\n$git_branch\\\n$character\"\"\"\nscan_timeout = 30\n")
	v, ok := p.FindValue(cfg, "format")
	if !ok || v != "$directory$git_branch$character" {
		t.Fatalf("FindValue format = %q (ok=%v), want %q", v, ok, "$directory$git_branch$character")
	}
	all := p.FindAll(cfg)
	if all["format"] != "$directory$git_branch$character" {
		t.Fatalf("FindAll format = %q", all["format"])
	}
	if all["add_newline"] != "true" || all["scan_timeout"] != "30" {
		t.Fatalf("neighbors corrupted: %#v", all)
	}
	keys := p.ListKeys(cfg)
	if len(keys) != 3 {
		t.Fatalf("ListKeys = %v, want 3 (body lines must be skipped)", keys)
	}
}

func TestMultilineWriteCollapsesBlock(t *testing.T) {
	p := &SectionParser{SplitKey: SplitKeyFirst}
	cfg := []byte("format = \"\"\"\n$directory\\\n$character\"\"\"\nscan_timeout = 30\n")
	out, err := p.SetValue(cfg, "format", "\"$directory$character\"")
	if err != nil {
		t.Fatal(err)
	}
	want := "format = \"$directory$character\"\nscan_timeout = 30\n"
	if string(out) != want {
		t.Fatalf("SetValue collapsed =\n%q\nwant\n%q", out, want)
	}
	v, _ := p.FindValue(out, "format")
	if v != "$directory$character" {
		t.Fatalf("re-read after write = %q", v)
	}
}

func TestMultilineLiteralAndInline(t *testing.T) {
	p := &SectionParser{SplitKey: SplitKeyFirst}
	// literal ''' is verbatim
	lit := []byte("x = '''\na\\b\n'''\n")
	if v, _ := p.FindValue(lit, "x"); v != "a\\b\n" {
		t.Fatalf("literal = %q", v)
	}
	// single-line triple quote
	one := []byte("y = \"\"\"hi\"\"\"\n")
	if v, _ := p.FindValue(one, "y"); v != "hi" {
		t.Fatalf("inline triple = %q", v)
	}
	// ordinary single-line value unaffected
	plain := []byte("z = \"hello\"\n")
	if v, _ := p.FindValue(plain, "z"); v != "hello" {
		t.Fatalf("plain = %q", v)
	}
}
