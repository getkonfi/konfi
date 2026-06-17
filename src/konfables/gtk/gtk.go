package gtk

import (
	"context"
	_ "embed"
	"os/exec"
	"strings"

	"github.com/getkonfi/konfi/konfables"
	"github.com/getkonfi/konfi/pkg"
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
		Icon:       "🎨",
		NerdIcon:   "", // nf-fa-paint_brush
		// no ReloadCmd / AutoReload: gtk reads settings at app start
	}
}

func (g *GTK) Name() string             { return "gtk" }
func (g *GTK) ConfigPath() string       { return g.Path }
func (g *GTK) Parser() konfables.Parser { return newParser() }
func (g *GTK) Schema() ([]byte, error)  { return schemaData, nil }

func (g *GTK) Version(ctx context.Context) (string, error) {
	for _, args := range [][]string{
		{"pkg-config", "--modversion", "gtk4"},
		{"pkg-config", "--modversion", "gtk+-3.0"},
		{"gtk-launch", "--version"},
	} {
		out, err := exec.CommandContext(ctx, args[0], args[1:]...).Output()
		if err != nil {
			continue
		}
		line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
		if v := pkg.ExtractSemver(line); v != "" {
			return v, nil
		}
		if line != "" {
			return line, nil
		}
	}
	out, err := exec.CommandContext(ctx, "gtk-launch", "--version").CombinedOutput()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if v := pkg.ExtractSemver(line); v != "" {
		return v, nil
	}
	return line, nil
}
