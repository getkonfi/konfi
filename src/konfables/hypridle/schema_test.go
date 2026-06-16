package hypridle

import (
	"testing"

	"github.com/getkonfi/konfi/pkg"
)

func TestSchemaLoads(t *testing.T) {
	s, err := pkg.LoadSchema(schemaData)
	if err != nil {
		t.Fatal(err)
	}
	if s.App != "hypridle" {
		t.Fatalf("app = %q, want hypridle", s.App)
	}
	if field := schemaField(t, s, listenersKey); field.Widget != "structlist" {
		t.Fatalf("listeners widget = %q, want structlist", field.Widget)
	}
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
