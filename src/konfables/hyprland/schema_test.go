package hyprland

import (
	"testing"

	"github.com/getkonfi/konfi/pkg"
)

func TestSchemaModelsTupleGapValues(t *testing.T) {
	s := loadSchema(t)

	for _, key := range []string{"general.gaps_in", "general.gaps_out"} {
		f := schemaField(t, s, key)
		if f.Type != "string" || f.Min != nil || f.Max != nil {
			t.Fatalf("%s = type %q min %v max %v, want string without bounds", key, f.Type, f.Min, f.Max)
		}
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
