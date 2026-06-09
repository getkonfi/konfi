package kitty

import (
	"testing"

	"github.com/eminert/konfi/pkg"
)

func TestSchemaModelsKittyEditorFidelityGaps(t *testing.T) {
	s := loadSchema(t)

	remote := schemaField(t, s, "allow_remote_control")
	if !hasOption(remote.Options, "socket") {
		t.Fatalf("allow_remote_control options missing socket: %v", remote.Options)
	}

	scrollback := schemaField(t, s, "scrollback_lines")
	if scrollback.Min == nil || *scrollback.Min != -1 {
		t.Fatalf("scrollback_lines min = %v, want -1", scrollback.Min)
	}

	padding := schemaField(t, s, "window_padding_width")
	if padding.Type != "string" || padding.Min != nil || padding.Max != nil {
		t.Fatalf("window_padding_width = type %q min %v max %v, want string without bounds", padding.Type, padding.Min, padding.Max)
	}
}

func loadSchema(t *testing.T) *pkg.Schema {
	t.Helper()
	s, err := pkg.LoadSchema(schemaData)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func schemaField(t *testing.T, s *pkg.Schema, key string) pkg.Field {
	t.Helper()
	for _, section := range s.Sections {
		for _, field := range section.Fields {
			if field.Key == key {
				return field
			}
		}
	}
	t.Fatalf("schema field %q not found", key)
	return pkg.Field{}
}

func hasOption(options []string, want string) bool {
	for _, opt := range options {
		if opt == want {
			return true
		}
	}
	return false
}
