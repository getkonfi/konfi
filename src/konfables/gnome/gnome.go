package gnome

import (
	_ "embed"

	"github.com/eminert/konfi/konfables"
	"github.com/eminert/konfi/pkg"
)

//go:embed schema.yaml
var schemaData []byte

type GNOME struct {
	pkg.Persister
}

func New(p pkg.Persister) *GNOME {
	return &GNOME{Persister: p}
}

func (g *GNOME) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:     "gnome",
		Binary:   "gsettings",
		Format:   "gnome",
		Icon:     "🐾",
		NerdIcon: "\uf30e", // nf-linux-gnome_old
	}
}

func (g *GNOME) Name() string             { return "gnome" }
func (g *GNOME) ConfigPath() string       { return "" }
func (g *GNOME) Parser() konfables.Parser { return newParser() }
func (g *GNOME) Schema() ([]byte, error)  { return schemaData, nil }
