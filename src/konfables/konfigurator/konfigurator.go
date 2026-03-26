package konfigurator

import (
	_ "embed"

	"github.com/emin/konfigurator/konfables"
	"github.com/emin/konfigurator/pkg"
)

//go:embed schema.yaml
var schemaData []byte

type Konfigurator struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Konfigurator {
	return &Konfigurator{FilePersister: p}
}

func (k *Konfigurator) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "konfigurator",
		Binary:     "",
		ConfigPath: k.Path,
		Format:     "yaml",
		Icon:       "⚙",
		NerdIcon:   "\uf013", // nerd font gear
	}
}

func (k *Konfigurator) Name() string            { return "konfigurator" }
func (k *Konfigurator) ConfigPath() string       { return k.Path }
func (k *Konfigurator) Parser() konfables.Parser { return newParser() }
func (k *Konfigurator) Schema() ([]byte, error)  { return schemaData, nil }
