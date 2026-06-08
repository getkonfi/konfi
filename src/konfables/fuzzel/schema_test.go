package fuzzel

import (
	"testing"

	"github.com/eminert/konfi/pkg"
)

func TestSchemaLoads(t *testing.T) {
	raw, err := New(pkg.NewFilePersister("")).Schema()
	if err != nil {
		t.Fatal(err)
	}
	s, err := pkg.LoadSchema(raw)
	if err != nil {
		t.Fatal(err)
	}
	if s.App != "fuzzel" {
		t.Fatalf("app = %q, want fuzzel", s.App)
	}
	if len(s.Sections) != 2 {
		t.Fatalf("sections = %d, want 2", len(s.Sections))
	}
}
