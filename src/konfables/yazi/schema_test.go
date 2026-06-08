package yazi

import (
	"testing"

	"github.com/eminert/konfi/pkg"
)

func TestSchema(t *testing.T) {
	s, err := pkg.LoadSchema(schemaData)
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}
	if s.App != "yazi" || s.Format != "toml" {
		t.Fatalf("schema app/format = %s/%s", s.App, s.Format)
	}
	keys := s.SchemaKeys()
	for _, want := range []string{
		"manager.show_hidden",
		"manager.sort_by",
		"manager.sort_sensitive",
		"preview.wrap",
		"preview.max_width",
		"opener.edit",
		"open.rules",
	} {
		if _, ok := keys[want]; !ok {
			t.Fatalf("schema missing %q", want)
		}
	}
}
