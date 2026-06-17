package dconf

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
		Icon:     "⚙️",
		NerdIcon: "\U000f164b", // nf-md-database_cog
	}
}

func (d *Dconf) Name() string             { return "dconf" }
func (d *Dconf) ConfigPath() string       { return "" }
func (d *Dconf) Parser() konfables.Parser { return newParser() }
func (d *Dconf) Schema() ([]byte, error)  { return schemaData, nil }

func (d *Dconf) Version(ctx context.Context) (string, error) {
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
