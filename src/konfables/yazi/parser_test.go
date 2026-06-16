package yazi

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/getkonfi/konfi/konfables"
	"github.com/getkonfi/konfi/pkg"
)

var testConfig = []byte(`# yazi config

[mgr]
show_hidden = false
sort_by = "alphabetical"
sort_sensitive = false

[preview]
wrap = "no"
max_width = 600

[opener]
edit = [
  { run = "${EDITOR:-vi} %s", desc = "$EDITOR", for = "unix", block = true },
]

[open]
rules = [
  # text
  { mime = "text/*", use = "edit" },
  { url = "*", use = "open" },
]
`)

func TestFindValue(t *testing.T) {
	p := newParser()

	tests := []struct {
		key  string
		want string
	}{
		{"manager.show_hidden", "false"},
		{"mgr.show_hidden", "false"},
		{"manager.sort_by", "alphabetical"},
		{"manager.sort_sensitive", "false"},
		{"preview.wrap", "no"},
		{"preview.max_width", "600"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := p.FindValue(testConfig, tt.key)
			if !ok {
				t.Fatalf("FindValue(%q) not found", tt.key)
			}
			if got != tt.want {
				t.Fatalf("FindValue(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestFindRawTOMLValue(t *testing.T) {
	p := newParser()

	edit, ok := p.FindValue(testConfig, "opener.edit")
	if !ok {
		t.Fatal("opener.edit not found")
	}
	if !strings.Contains(edit, `${EDITOR:-vi}`) || !strings.Contains(edit, "block = true") {
		t.Fatalf("opener.edit = %q", edit)
	}

	rules, ok := p.FindValue(testConfig, "open.rules")
	if !ok {
		t.Fatal("open.rules not found")
	}
	if !strings.Contains(rules, `mime = "text/*"`) || !strings.Contains(rules, `url = "*"`) {
		t.Fatalf("open.rules = %q", rules)
	}
}

func TestFindLine(t *testing.T) {
	p := newParser()

	tests := []struct {
		key  string
		want int
	}{
		{"manager.show_hidden", 3},
		{"preview.wrap", 8},
		{"opener.edit", 12},
		{"open.rules", 17},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := p.FindLine(testConfig, tt.key)
			if !ok {
				t.Fatalf("FindLine(%q) not found", tt.key)
			}
			if got != tt.want {
				t.Fatalf("FindLine(%q) = %d, want %d", tt.key, got, tt.want)
			}
		})
	}
}

func TestSetValue(t *testing.T) {
	p := newParser()

	out, err := p.SetValue(testConfig, "manager.show_hidden", "true")
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := p.FindValue(out, "manager.show_hidden"); !ok || got != "true" {
		t.Fatalf("manager.show_hidden = %q ok=%v", got, ok)
	}
	if bytes.Contains(out, []byte("[manager]")) {
		t.Fatal("manager alias wrote [manager], want [mgr]")
	}

	out, err = p.SetValue(out, "preview.max_width", "1024")
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := p.FindValue(out, "preview.max_width"); !ok || got != "1024" {
		t.Fatalf("preview.max_width = %q ok=%v", got, ok)
	}
}

func TestSetRawTOMLValue(t *testing.T) {
	p := newParser()
	value := `[{ run = "nvim %s", block = true, for = "unix" }]`

	out, err := p.SetValue(testConfig, "opener.edit", strconv.Quote(value))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.FindValue(out, "opener.edit")
	if !ok {
		t.Fatal("opener.edit not found")
	}
	if got != value {
		t.Fatalf("opener.edit = %q, want %q", got, value)
	}
	if bytes.Contains(out, []byte(`\"nvim`)) {
		t.Fatalf("raw toml value was escaped:\n%s", out)
	}
}

func TestWriteFieldOpenRulesDefaultWritesRawArray(t *testing.T) {
	p := newParser()
	field := pkg.Field{Key: "open.rules", Type: "string", Widget: "rawtoml", Default: "[]"}

	out, err := konfables.WriteField(p, []byte("[open]\n"), field, field.Default, "toml")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte("rules = []")) || bytes.Contains(out, []byte(`rules = "[]"`)) {
		t.Fatalf("open.rules default was not written as raw TOML array:\n%s", out)
	}
}

func TestDeleteRawTOMLValue(t *testing.T) {
	p := newParser()

	out, err := p.DeleteKey(testConfig, "open.rules")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.FindValue(out, "open.rules"); ok {
		t.Fatal("open.rules still found")
	}
	if bytes.Contains(out, []byte(`mime = "text/*"`)) {
		t.Fatalf("open.rules body still present:\n%s", out)
	}
}

func TestListKeys(t *testing.T) {
	p := newParser()
	keys := p.ListKeys(testConfig)
	found := make(map[string]bool, len(keys))
	for _, key := range keys {
		found[key] = true
	}
	for _, want := range []string{
		"manager.show_hidden",
		"manager.sort_by",
		"manager.sort_sensitive",
		"preview.wrap",
		"preview.max_width",
		"opener.edit",
		"open.rules",
	} {
		if !found[want] {
			t.Fatalf("ListKeys missing %q in %v", want, keys)
		}
	}
	for _, unexpected := range []string{
		"opener.{ run",
		"open.{ mime",
		"open.{ url",
	} {
		if found[unexpected] {
			t.Fatalf("ListKeys included raw TOML body key %q in %v", unexpected, keys)
		}
	}
}

func TestFindAll(t *testing.T) {
	p := newParser()
	all := p.FindAll(testConfig)

	if all["manager.show_hidden"] != "false" {
		t.Fatalf("manager.show_hidden = %q", all["manager.show_hidden"])
	}
	if !strings.Contains(all["opener.edit"], `${EDITOR:-vi}`) {
		t.Fatalf("opener.edit = %q", all["opener.edit"])
	}
	if !strings.Contains(all["open.rules"], `mime = "text/*"`) {
		t.Fatalf("open.rules = %q", all["open.rules"])
	}
	for _, unexpected := range []string{
		"opener.{ run",
		"open.{ mime",
		"open.{ url",
	} {
		if _, ok := all[unexpected]; ok {
			t.Fatalf("FindAll included raw TOML body key %q", unexpected)
		}
	}
}
