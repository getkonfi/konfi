package dconf

import (
	_ "embed"

	"github.com/emin/konfigurator/konfables"
	"github.com/emin/konfigurator/pkg"
)

//go:embed schema.yaml
var schemaData []byte

type Dconf struct {
	pkg.Persister
}

func New(p pkg.Persister) *Dconf {
	return &Dconf{Persister: p}
}

func (d *Dconf) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:     "dconf",
		Binary:   "dconf",
		Format:   "dconf",
		Icon:     "⚙",
		NerdIcon: "\uf013", // nf-fa-cog
	}
}

func (d *Dconf) Name() string             { return "dconf" }
func (d *Dconf) ConfigPath() string       { return "" }
func (d *Dconf) Parser() konfables.Parser { return newParser() }
func (d *Dconf) Schema() ([]byte, error)  { return schemaData, nil }
