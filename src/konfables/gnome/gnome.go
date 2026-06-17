package gnome

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
		NerdIcon: "\uf361", // nf-linux-gnome
	}
}

func (g *GNOME) Name() string             { return "gnome" }
func (g *GNOME) ConfigPath() string       { return "" }
func (g *GNOME) Parser() konfables.Parser { return newParser() }
func (g *GNOME) Schema() ([]byte, error)  { return schemaData, nil }

func (g *GNOME) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "pkg-config", "--modversion", "gsettings-desktop-schemas").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if v := pkg.ExtractSemver(line); v != "" {
		return v, nil
	}
	return line, nil
}
