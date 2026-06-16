package gtk

import (
	_ "embed"

	"github.com/getkonfi/konfi/konfables"
)

//go:embed schema.yaml
var schemaData []byte

// GTK edits the shared GTK appearance settings (theme, icons, cursor, font)
// stored in gtk-3.0/settings.ini and gtk-4.0/settings.ini.
type GTK struct {
	*MirrorPersister
}

func New(p *MirrorPersister) *GTK {
	return &GTK{MirrorPersister: p}
}

func (g *GTK) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "gtk",
		Binary:     "gtk-launch",
		ConfigPath: g.Path,
		Format:     "ini",
		Icon:       "",
		NerdIcon:   "", // nf-fa-paint_brush
		// no ReloadCmd / AutoReload: gtk reads settings at app start
	}
}

func (g *GTK) Name() string             { return "gtk" }
func (g *GTK) ConfigPath() string       { return g.Path }
func (g *GTK) Parser() konfables.Parser { return newParser() }
func (g *GTK) Schema() ([]byte, error)  { return schemaData, nil }
