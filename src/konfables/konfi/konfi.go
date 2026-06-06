package konfi

import (
	_ "embed"

	"github.com/eminert/konfi/konfables"
	"github.com/eminert/konfi/pkg"
)

//go:embed schema.yaml
var schemaData []byte

type Konfi struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Konfi {
	return &Konfi{FilePersister: p}
}

func (k *Konfi) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "konfi",
		Binary:     "",
		ConfigPath: k.Path,
		Format:     "yaml",
		Icon:       "⚙",
		NerdIcon:   "\uf013", // nerd font gear
	}
}

func (k *Konfi) Name() string             { return "konfi" }
func (k *Konfi) ConfigPath() string       { return k.Path }
func (k *Konfi) Parser() konfables.Parser { return newParser() }
func (k *Konfi) Schema() ([]byte, error)  { return schemaData, nil }
