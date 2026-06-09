package ghostty

import (
	"testing"

	"github.com/eminert/konfi/pkg"
)

func TestSchemaModelsLocal131Gaps(t *testing.T) {
	s := loadSchema(t)

	for _, key := range []string{"font-style", "font-style-bold", "font-style-italic", "font-style-bold-italic"} {
		f := schemaField(t, s, key)
		if !hasOption(f.Options, "default") {
			t.Fatalf("%s options missing default: %v", key, f.Options)
		}
	}

	synthetic := schemaField(t, s, "font-synthetic-style")
	if synthetic.Type != "multi" {
		t.Fatalf("font-synthetic-style type = %q, want multi", synthetic.Type)
	}
	for _, opt := range []string{"true", "false", "no-bold", "no-italic"} {
		if !hasOption(synthetic.Options, opt) {
			t.Fatalf("font-synthetic-style options missing %q: %v", opt, synthetic.Options)
		}
	}

	freetype := schemaField(t, s, "freetype-load-flags")
	for _, opt := range []string{"true", "false"} {
		if !hasOption(freetype.Options, opt) {
			t.Fatalf("freetype-load-flags options missing %q: %v", opt, freetype.Options)
		}
	}

	keybind := schemaField(t, s, "keybind")
	action := schemaPart(t, keybind, "action")
	if action.Type != "string" || len(action.Options) != 0 {
		t.Fatalf("keybind action part = type %q options %v, want unconstrained string", action.Type, action.Options)
	}

	scroll := schemaField(t, s, "mouse-scroll-multiplier")
	if scroll.Type != "string" || scroll.Min != nil || scroll.Max != nil {
		t.Fatalf("mouse-scroll-multiplier = type %q min %v max %v, want string without bounds", scroll.Type, scroll.Min, scroll.Max)
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

func schemaPart(t *testing.T, f pkg.Field, name string) pkg.FieldPart {
	t.Helper()
	for _, part := range f.ItemSchema {
		if part.Name == name {
			return part
		}
	}
	t.Fatalf("schema part %q not found in %s", name, f.Key)
	return pkg.FieldPart{}
}

func hasOption(options []string, want string) bool {
	for _, opt := range options {
		if opt == want {
			return true
		}
	}
	return false
}
