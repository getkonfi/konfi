package fuzzel

import (
	"bytes"
	"reflect"
	"testing"
)

var sampleConfig = []byte(`# fuzzel
font=monospace
dpi-aware=auto
terminal=foot -e
prompt="> "
icon-theme=Papirus
width=30
tabs=8

[colors]
background=fdf6e3ff
text=657b83ff
match=cb4b16ff
selection=eee8d5ff
selection-text=586e75ff
border=002b36ff
`)

func TestFindValue(t *testing.T) {
	p := newParser()

	tests := []struct {
		key  string
		want string
		ok   bool
	}{
		{"font", "monospace", true},
		{"dpi-aware", "auto", true},
		{"prompt", "> ", true},
		{"colors.background", "fdf6e3ff", true},
		{"colors.selection-text", "586e75ff", true},
		{"missing", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := p.FindValue(sampleConfig, tt.key)
			if ok != tt.ok {
				t.Fatalf("FindValue(%q) ok = %v, want %v", tt.key, ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("FindValue(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestFindValueUsesMainSectionAlias(t *testing.T) {
	p := newParser()
	data := []byte("[main]\nfont=JetBrains Mono:size=11\nwidth=45\n\n[colors]\nborder=89b4faff\n")

	got, ok := p.FindValue(data, "font")
	if !ok || got != "JetBrains Mono:size=11" {
		t.Fatalf("FindValue(font) = %q, %v", got, ok)
	}
	line, ok := p.FindLine(data, "width")
	if !ok || line != 2 {
		t.Fatalf("FindLine(width) = %d, %v", line, ok)
	}
}

func TestSetValue(t *testing.T) {
	p := newParser()

	out, err := p.SetValue(sampleConfig, "width", "50")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte("width=50\n")) {
		t.Fatalf("SetValue(width) did not preserve existing line style:\n%s", out)
	}

	out, err = p.SetValue(out, "colors.selection", "313244ff")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.FindValue(out, "colors.selection")
	if !ok || got != "313244ff" {
		t.Fatalf("colors.selection = %q, %v", got, ok)
	}

	out, err = p.SetValue(out, "colors.selection-match", "f38ba8ff")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte("selection-match = f38ba8ff\n")) {
		t.Fatalf("SetValue(colors.selection-match) did not insert ini assignment:\n%s", out)
	}
}

func TestSetValueReusesMainSection(t *testing.T) {
	p := newParser()
	data := []byte("[main]\nfont=monospace\n\n[colors]\nbackground=fdf6e3ff\n")

	out, err := p.SetValue(data, "font", "Iosevka:size=12")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte("[main]\nfont=Iosevka:size=12\n")) {
		t.Fatalf("SetValue(font) should update [main] in place:\n%s", out)
	}
	if bytes.Contains(out, []byte("\nfont = Iosevka:size=12\n[main]")) {
		t.Fatalf("SetValue(font) added duplicate top-level key:\n%s", out)
	}
}

func TestSetValueAddsMissingKeyToMainSection(t *testing.T) {
	p := newParser()
	data := []byte("[main]\nfont=monospace\n\n[colors]\nbackground=fdf6e3ff\n")

	out, err := p.SetValue(data, "width", "45")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte("width = 45\n[colors]")) {
		t.Fatalf("SetValue(width) should add to [main]:\n%s", out)
	}
}

func TestDeleteKey(t *testing.T) {
	p := newParser()

	out, err := p.DeleteKey(sampleConfig, "colors.match")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.FindValue(out, "colors.match"); ok {
		t.Fatal("colors.match should be deleted")
	}
	if got, ok := p.FindValue(out, "colors.text"); !ok || got != "657b83ff" {
		t.Fatalf("colors.text should survive, got %q, %v", got, ok)
	}
}

func TestListKeysNormalizesMainSection(t *testing.T) {
	p := newParser()
	data := []byte("[main]\nfont=monospace\n\n[colors]\nbackground=fdf6e3ff\n")

	got := p.ListKeys(data)
	want := []string{"font", "colors.background"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListKeys() = %#v, want %#v", got, want)
	}
}
