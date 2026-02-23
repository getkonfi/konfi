package konfigurator

import (
	_ "embed"

	"github.com/emin/konfigurator/konfables"
	"github.com/emin/konfigurator/setup/cst"
)

//go:embed schema.yaml
var schemaData []byte

type Konfigurator struct{}

func New() *Konfigurator { return &Konfigurator{} }

func (k *Konfigurator) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "konfigurator",
		Binary:     "",
		ConfigPath: cst.ConfigFilePath(),
		Format:     "yaml",
		Icon:       "⚙",
		NerdIcon:   "\uf013", // nerd font gear
	}
}

func (k *Konfigurator) Name() string            { return "konfigurator" }
func (k *Konfigurator) ConfigPath() string       { return cst.ConfigFilePath() }
func (k *Konfigurator) Parser() konfables.Parser { return &parser{} }
func (k *Konfigurator) Schema() ([]byte, error)  { return schemaData, nil }
