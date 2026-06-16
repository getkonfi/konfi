package waybar

import (
	"strings"
	"testing"

	"github.com/getkonfi/konfi/pkg"
)

func TestSchemaLoadsWaybarFields(t *testing.T) {
	s, err := pkg.LoadSchema(schemaData)
	if err != nil {
		t.Fatal(err)
	}
	if s.App != "waybar" {
		t.Fatalf("app = %q, want waybar", s.App)
	}
	if s.Format != "json" {
		t.Fatalf("format = %q, want json", s.Format)
	}

	keys := s.SchemaKeys()
	for _, key := range []string{
		"position",
		"layer",
		"height",
		"modules-left",
		"modules-center",
		"modules-right",
		"clock.format",
		"battery.format",
		"network.format-wifi",
		"pulseaudio.format",
		"tray.spacing",
	} {
		if _, ok := keys[key]; !ok {
			t.Fatalf("schema missing key %q", key)
		}
	}
}

func TestSchemaNotesJSONCNormalization(t *testing.T) {
	s, err := pkg.LoadSchema(schemaData)
	if err != nil {
		t.Fatal(err)
	}
	for _, hint := range s.Hints {
		if strings.Contains(hint, "jsonc") && strings.Contains(hint, "normalize") {
			return
		}
	}
	t.Fatal("schema should note jsonc write normalization")
}
