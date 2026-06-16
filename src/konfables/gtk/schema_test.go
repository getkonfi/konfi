package gtk

import (
	"testing"

	"github.com/getkonfi/konfi/pkg"
)

func TestSchemaKeepsGTKFontNameAsFullPangoString(t *testing.T) {
	s := loadSchema(t)

	font := schemaField(t, s, "Settings.gtk-font-name")
	if font.Type != "string" || font.Widget != "" {
		t.Fatalf("gtk-font-name = type %q widget %q, want plain string", font.Type, font.Widget)
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
		for i := range section.Fields {
			field := section.Fields[i]
			if field.Key == key {
				return field
			}
		}
	}
	t.Fatalf("schema field %q not found", key)
	return pkg.Field{}
}
