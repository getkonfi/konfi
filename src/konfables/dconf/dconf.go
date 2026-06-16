package dconf

import (
	_ "embed"

	"github.com/getkonfi/konfi/konfables"
	"github.com/getkonfi/konfi/pkg"
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
		NerdIcon: "\U000f164b", // nf-md-database_cog
	}
}

func (d *Dconf) Name() string             { return "dconf" }
func (d *Dconf) ConfigPath() string       { return "" }
func (d *Dconf) Parser() konfables.Parser { return newParser() }
func (d *Dconf) Schema() ([]byte, error)  { return schemaData, nil }
